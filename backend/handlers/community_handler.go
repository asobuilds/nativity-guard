package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"security-solution/config"
	"security-solution/models"
)

// Community responses explicitly allow public fields; never serialize a User model.
func communityAuthor(user models.User) gin.H {
	return gin.H{"name": strings.TrimSpace(user.FirstName + " " + user.LastName)}
}

func forumPostResponse(post models.ForumPost, includeReplies bool) gin.H {
	result := gin.H{
		"id": post.ID, "unitId": post.UnitID, "title": post.Title,
		"content": post.Content, "category": post.Category,
		"author": communityAuthor(post.Author), "createdAt": post.CreatedAt,
		"isPinned": post.IsPinned, "isLocked": post.IsLocked,
		"replyCount": post.ReplyCount, "viewCount": post.ViewCount,
	}
	if includeReplies {
		replies := make([]gin.H, 0, len(post.Replies))
		for _, reply := range post.Replies {
			if reply.Status != "published" { continue }
			replies = append(replies, gin.H{"id": reply.ID, "content": reply.Content, "author": communityAuthor(reply.Author), "createdAt": reply.CreatedAt})
		}
		result["replies"] = replies
	}
	return result
}

func communityAnnouncementResponse(item models.CommunityAnnouncement) gin.H {
	return gin.H{"id": item.ID, "unitId": item.UnitID, "title": item.Title,
		"content": item.Content, "type": item.Type, "priority": item.Priority,
		"createdAt": item.CreatedAt, "publishedAt": item.PublishedAt, "expiresAt": item.ExpiresAt}
}

func communityEventResponse(item models.CommunityEvent) gin.H {
	return gin.H{"id": item.ID, "unitId": item.UnitID, "title": item.Title,
		"description": item.Description, "location": item.Location,
		"eventDate": item.EventDate, "endDate": item.EndDate, "type": item.Type,
		"attendeeCount": item.AttendeeCount, "maxAttendees": item.MaxAttendees,
		"createdAt": item.CreatedAt}
}

// CreateForumPost creates a new forum post
func CreateForumPost(c *gin.Context) {
	var input struct {
		UnitID   string `json:"unitId"`
		Title    string `json:"title" binding:"required"`
		Content  string `json:"content" binding:"required"`
		Category string `json:"category"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	var unitID *uuid.UUID
	if input.UnitID != "" {
		parsed, err := uuid.Parse(input.UnitID)
		if err == nil {
			unitID = &parsed
			// Verify user belongs to this unit
			if userObj.UnitID == nil || *userObj.UnitID != *unitID {
				c.JSON(http.StatusForbidden, gin.H{"error": "You don't belong to this unit"})
				return
			}
		}
	}

	if input.Category == "" {
		input.Category = "general"
	}

	post := models.ForumPost{
		UnitID:   unitID,
		Title:    input.Title,
		Content:  input.Content,
		Category: input.Category,
		AuthorID: userObj.ID,
		Status:   "published",
	}

	if err := config.DB.Create(&post).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create post"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Post created successfully",
		"post":    forumPostResponse(post, false),
	})
}

// GetForumPosts gets all forum posts
func GetForumPosts(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	var posts []models.ForumPost
	query := config.DB.Preload("Author", func(db *gorm.DB) *gorm.DB { return db.Select("id", "first_name", "last_name") }).Where("status = ?", "published")

	// Filter by role
	if userObj.Role == "citizen" {
		// Citizens see public posts (no unit assigned) and their own unit posts
		if userObj.UnitID != nil {
			query = query.Where("unit_id IS NULL OR unit_id = ?", userObj.UnitID)
		} else {
			query = query.Where("unit_id IS NULL")
		}
	} else if userObj.Role == "officer" && userObj.UnitID != nil {
		// Officers see unit posts and public posts
		query = query.Where("unit_id IS NULL OR unit_id = ?", userObj.UnitID)
	} else if userObj.Role == "unit_admin" && userObj.UnitID != nil {
		// Admins see all posts in their unit
		query = query.Where("unit_id IS NULL OR unit_id = ?", userObj.UnitID)
	} else if userObj.Role == "super_admin" {
		// Super admin sees all
	}

	if err := query.Order("is_pinned DESC, created_at DESC").Find(&posts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch posts"})
		return
	}

	results := make([]gin.H, 0, len(posts))
	for _, post := range posts { results = append(results, forumPostResponse(post, false)) }
	c.JSON(http.StatusOK, gin.H{"posts": results})
}

// GetForumPostByID gets a specific forum post
func GetForumPostByID(c *gin.Context) {
	id := c.Param("id")
	postID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid post ID"})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	var post models.ForumPost
	if err := config.DB.Preload("Author", func(db *gorm.DB) *gorm.DB { return db.Select("id", "first_name", "last_name") }).Preload("Replies", "status = ?", "published").Preload("Replies.Author", func(db *gorm.DB) *gorm.DB { return db.Select("id", "first_name", "last_name") }).First(&post, "id = ? AND status = ?", postID, "published").Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
		return
	}

	// Check access
	if post.UnitID != nil && userObj.Role != "super_admin" {
		if userObj.UnitID == nil || *userObj.UnitID != *post.UnitID {
			c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to view this post"})
			return
		}
	}

	// Increment view count
	post.ViewCount++
	config.DB.Save(&post)

	c.JSON(http.StatusOK, gin.H{
		"post": forumPostResponse(post, true),
	})
}

// CreateForumReply creates a reply to a forum post
func CreateForumReply(c *gin.Context) {
	var input struct {
		PostID  string `json:"postId" binding:"required"`
		Content string `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	postID, err := uuid.Parse(input.PostID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid post ID"})
		return
	}

	var post models.ForumPost
	if err := config.DB.First(&post, "id = ?", postID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
		return
	}

	if post.IsLocked {
		c.JSON(http.StatusForbidden, gin.H{"error": "This post is locked"})
		return
	}

	// Check access
	if post.UnitID != nil && userObj.Role != "super_admin" {
		if userObj.UnitID == nil || *userObj.UnitID != *post.UnitID {
			c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to reply to this post"})
			return
		}
	}

	reply := models.ForumReply{
		PostID:   postID,
		Content:  input.Content,
		AuthorID: userObj.ID,
		Status:   "published",
	}

	if err := config.DB.Create(&reply).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create reply"})
		return
	}

	// Update reply count
	post.ReplyCount++
	now := time.Now()
	post.LastReplyAt = &now
	config.DB.Save(&post)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Reply created successfully",
		"reply":   gin.H{"id": reply.ID, "postId": reply.PostID, "content": reply.Content, "createdAt": reply.CreatedAt},
	})
}

// CreateCommunityAnnouncement creates a community announcement
func CreateCommunityAnnouncement(c *gin.Context) {
	var input struct {
		UnitID    string `json:"unitId"`
		Title     string `json:"title" binding:"required"`
		Content   string `json:"content" binding:"required"`
		Type      string `json:"type"`
		Priority  string `json:"priority"`
		ExpiresAt string `json:"expiresAt"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	// Only admins can create announcements
	if userObj.Role != "super_admin" && userObj.Role != "unit_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins can create announcements"})
		return
	}

	var unitID *uuid.UUID
	if input.UnitID != "" {
		parsed, err := uuid.Parse(input.UnitID)
		if err == nil {
			unitID = &parsed
			if userObj.Role == "unit_admin" {
				if userObj.UnitID == nil || *userObj.UnitID != *unitID {
					c.JSON(http.StatusForbidden, gin.H{"error": "You can only create announcements for your unit"})
					return
				}
			}
		}
	} else {
		// If no unit specified, use user's unit
		if userObj.UnitID != nil {
			unitID = userObj.UnitID
		}
	}

	var expiresAt *time.Time
	if input.ExpiresAt != "" {
		parsed, err := time.Parse(time.RFC3339, input.ExpiresAt)
		if err == nil {
			expiresAt = &parsed
		}
	}

	if input.Type == "" {
		input.Type = "general"
	}
	if input.Priority == "" {
		input.Priority = "normal"
	}

	announcement := models.CommunityAnnouncement{
		UnitID:      unitID,
		Title:       input.Title,
		Content:     input.Content,
		Type:        input.Type,
		Priority:    input.Priority,
		AuthorID:    userObj.ID,
		Status:      "published",
		ExpiresAt:   expiresAt,
		PublishedAt: time.Now(),
	}

	if err := config.DB.Create(&announcement).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create announcement"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":      "Announcement created successfully",
		"announcement": communityAnnouncementResponse(announcement),
	})
}

// GetCommunityAnnouncements gets community announcements
func GetCommunityAnnouncements(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	var announcements []models.CommunityAnnouncement
	query := config.DB.Where("status = ?", "published")

	// Filter by role
	if userObj.Role == "citizen" {
		if userObj.UnitID != nil {
			query = query.Where("unit_id IS NULL OR unit_id = ?", userObj.UnitID)
		} else {
			query = query.Where("unit_id IS NULL")
		}
	} else if userObj.Role == "officer" && userObj.UnitID != nil {
		query = query.Where("unit_id IS NULL OR unit_id = ?", userObj.UnitID)
	} else if userObj.Role == "unit_admin" && userObj.UnitID != nil {
		query = query.Where("unit_id IS NULL OR unit_id = ?", userObj.UnitID)
	}

	if err := query.Order("priority DESC, created_at DESC").Find(&announcements).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch announcements"})
		return
	}

	results := make([]gin.H, 0, len(announcements))
	for _, item := range announcements { results = append(results, communityAnnouncementResponse(item)) }
	c.JSON(http.StatusOK, gin.H{"announcements": results})
}

// CreateCommunityEvent creates a community event
func CreateCommunityEvent(c *gin.Context) {
	var input struct {
		UnitID       string  `json:"unitId"`
		Title        string  `json:"title" binding:"required"`
		Description  string  `json:"description" binding:"required"`
		Location     string  `json:"location" binding:"required"`
		Latitude     float64 `json:"latitude"`
		Longitude    float64 `json:"longitude"`
		EventDate    string  `json:"eventDate" binding:"required"`
		EndDate      string  `json:"endDate" binding:"required"`
		Type         string  `json:"type" binding:"required"`
		MaxAttendees int     `json:"maxAttendees"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	// Only admins can create events
	if userObj.Role != "super_admin" && userObj.Role != "unit_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins can create events"})
		return
	}

	var unitID *uuid.UUID
	if input.UnitID != "" {
		parsed, err := uuid.Parse(input.UnitID)
		if err == nil {
			unitID = &parsed
			if userObj.Role == "unit_admin" {
				if userObj.UnitID == nil || *userObj.UnitID != *unitID {
					c.JSON(http.StatusForbidden, gin.H{"error": "You can only create events for your unit"})
					return
				}
			}
		}
	} else {
		if userObj.UnitID != nil {
			unitID = userObj.UnitID
		}
	}

	eventDate, err := time.Parse(time.RFC3339, input.EventDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event date"})
		return
	}

	var endDate *time.Time
	if input.EndDate != "" {
		parsed, err := time.Parse(time.RFC3339, input.EndDate)
		if err == nil {
			endDate = &parsed
		}
	}

	if input.Type == "" {
		input.Type = "meeting"
	}

	event := models.CommunityEvent{
		UnitID:       unitID,
		Title:        input.Title,
		Description:  input.Description,
		Location:     input.Location,
		Latitude:     input.Latitude,
		Longitude:    input.Longitude,
		EventDate:    eventDate,
		EndDate:      endDate,
		Type:         input.Type,
		Status:       "upcoming",
		MaxAttendees: input.MaxAttendees,
		AuthorID:     userObj.ID,
	}

	if err := config.DB.Create(&event).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create event"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Event created successfully",
		"event":   communityEventResponse(event),
	})
}

// GetCommunityEvents gets community events
func GetCommunityEvents(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	var events []models.CommunityEvent
	query := config.DB.Where("status = ?", "upcoming")

	if userObj.Role == "citizen" {
		if userObj.UnitID != nil {
			query = query.Where("unit_id IS NULL OR unit_id = ?", userObj.UnitID)
		} else {
			query = query.Where("unit_id IS NULL")
		}
	} else if userObj.Role == "officer" && userObj.UnitID != nil {
		query = query.Where("unit_id IS NULL OR unit_id = ?", userObj.UnitID)
	} else if userObj.Role == "unit_admin" && userObj.UnitID != nil {
		query = query.Where("unit_id IS NULL OR unit_id = ?", userObj.UnitID)
	}

	if err := query.Order("event_date ASC").Find(&events).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch events"})
		return
	}

	results := make([]gin.H, 0, len(events))
	for _, item := range events { results = append(results, communityEventResponse(item)) }
	c.JSON(http.StatusOK, gin.H{"events": results})
}

// RSVPToEvent RSVP to an event
func RSVPToEvent(c *gin.Context) {
	eventID := c.Param("id")
	id, err := uuid.Parse(eventID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID"})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	var event models.CommunityEvent
	if err := config.DB.First(&event, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	// Check if already RSVP'd
	var existing models.EventAttendee
	if err := config.DB.Where("event_id = ? AND user_id = ?", id, userObj.ID).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You have already RSVP'd to this event"})
		return
	}

	// Check capacity
	if event.MaxAttendees > 0 && event.AttendeeCount >= event.MaxAttendees {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Event is full"})
		return
	}

	attendee := models.EventAttendee{
		EventID: id,
		UserID:  userObj.ID,
		Status:  "registered",
	}

	if err := config.DB.Create(&attendee).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to RSVP"})
		return
	}

	event.AttendeeCount++
	config.DB.Save(&event)

	c.JSON(http.StatusOK, gin.H{
		"message": "RSVP successful",
	})
}
