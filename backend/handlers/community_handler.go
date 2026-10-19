package handlers

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

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
		"scope": post.GeoPlace.Scope, "place": geoPlaceResponse(post.GeoPlace),
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

// CreateForumPost creates a new forum post.
//
// The post's place is resolved server-side (decision D-012): the client chooses
// an audience scope and may attach its current coordinates, but never place
// names. Publishing to a scope the caller's place cannot express is rejected,
// so an "LGA" post always carries a real LGA.
func CreateForumPost(c *gin.Context) {
	var input struct {
		UnitID    string   `json:"unitId"`
		Title     string   `json:"title" binding:"required"`
		Content   string   `json:"content" binding:"required"`
		Category  string   `json:"category"`
		Scope     string   `json:"scope"`
		Latitude  *float64 `json:"latitude"`
		Longitude *float64 `json:"longitude"`
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
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid unit ID"})
			return
		}
		{
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

	scope := strings.ToLower(strings.TrimSpace(input.Scope))
	if scope == "" {
		scope = defaultScopeFor(resolveViewerPlace(userObj, input.Latitude, input.Longitude))
	}
	if !validScope(scope) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "scope must be one of: locality, lga, state, country, world"})
		return
	}

	place := resolveViewerPlace(userObj, input.Latitude, input.Longitude)
	if !scopeSatisfiedBy(scope, place) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "your location is not detailed enough to post to " + scope + "; share your location and try again",
		})
		return
	}

	post := models.ForumPost{
		UnitID:   unitID,
		GeoPlace: newPlace(scope, place, input.Latitude, input.Longitude),
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

// feedCursor is a keyset position in the feed. The feed is ordered
// `is_pinned DESC, created_at DESC, id DESC`, so the cursor carries all three: a
// created_at-only cursor would skip or repeat posts that share a timestamp, and
// would break across the pinned/unpinned boundary.
type feedCursor struct {
	Pinned    bool
	CreatedAt time.Time
	ID        uuid.UUID
}

func encodeFeedCursor(cursor feedCursor) string {
	raw := fmt.Sprintf("%t|%s|%s",
		cursor.Pinned,
		cursor.CreatedAt.UTC().Format(time.RFC3339Nano),
		cursor.ID.String(),
	)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// decodeFeedCursor returns ok=false for anything that is not a cursor this
// handler produced, so a malformed value is a 400 rather than a silent page 1.
func decodeFeedCursor(value string) (feedCursor, bool) {
	decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(value))
	if err != nil {
		return feedCursor{}, false
	}
	parts := strings.SplitN(string(decoded), "|", 3)
	if len(parts) != 3 {
		return feedCursor{}, false
	}
	pinned, err := strconv.ParseBool(parts[0])
	if err != nil {
		return feedCursor{}, false
	}
	createdAt, err := time.Parse(time.RFC3339Nano, parts[1])
	if err != nil {
		return feedCursor{}, false
	}
	id, err := uuid.Parse(parts[2])
	if err != nil {
		return feedCursor{}, false
	}
	return feedCursor{Pinned: pinned, CreatedAt: createdAt, ID: id}, true
}

// feedCursorCondition is the SQL for "strictly after this row" under the feed's
// ordering. `is_pinned` is a boolean and Postgres orders false < true, so DESC
// puts true first and `<` on the tuple walks the feed forward.
//
// The existing idx_forum_posts_feed_order (status, is_pinned DESC, created_at
// DESC) still serves this scan; the id tiebreak is a cheap residual filter rather
// than a new index, because created_at collisions are rare.
func feedCursorCondition(cursor feedCursor) (string, []any) {
	return "(is_pinned < ? OR (is_pinned = ? AND (created_at < ? OR (created_at = ? AND id < ?))))",
		[]any{cursor.Pinned, cursor.Pinned, cursor.CreatedAt, cursor.CreatedAt, cursor.ID}
}

// GetForumPosts returns the caller's location-scoped feed.
//
// `scope` widens how much of the *viewer's own* jurisdiction is returned
// (default lga). It can never reach another place: every branch of the WHERE
// clause pins the post's place to the viewer's resolved place (decision D-013).
// Super admins moderate the whole platform and are not bounded.
//
// Paging is keyset: send back the `nextCursor` from the previous page as
// `?cursor=`. `offset` is still accepted for older clients but is ignored when a
// cursor is present. Offset paging is unstable under concurrent inserts and slows
// down as the feed grows, which is why the cursor exists.
func GetForumPosts(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj := user.(*models.User)

	scope := strings.ToLower(strings.TrimSpace(c.Query("scope")))
	if scope == "" {
		scope = ScopeLGA
	}
	maxRank, ok := scopeRank(scope)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "scope must be one of: locality, lga, state, country, world"})
		return
	}

	limit := 20
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 50 {
			limit = n
		}
	}

	var cursor *feedCursor
	if raw := strings.TrimSpace(c.Query("cursor")); raw != "" {
		parsed, valid := decodeFeedCursor(raw)
		if !valid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid cursor"})
			return
		}
		cursor = &parsed
	}
	offset := 0
	if cursor == nil {
		if raw := strings.TrimSpace(c.Query("offset")); raw != "" {
			if n, err := strconv.Atoi(raw); err == nil && n > 0 {
				offset = n
			}
		}
	}

	var posts []models.ForumPost
	query := config.DB.
		Preload("Author", func(db *gorm.DB) *gorm.DB { return db.Select("id", "first_name", "last_name") }).
		Where("status = ?", "published")

	if userObj.Role != "super_admin" {
		lat, lng := parseCoords(c)
		place := resolveViewerPlace(userObj, lat, lng)
		condition, args := scopeWindowCondition(place, maxRank)
		if userObj.UnitID != nil {
			// Keep the caller's own unit's posts in view in addition to the
			// geographic window.
			query = query.Where("(unit_id = ? OR "+condition+")", append([]any{*userObj.UnitID}, args...)...)
		} else {
			query = query.Where(condition, args...)
		}
	}

	if cursor != nil {
		condition, args := feedCursorCondition(*cursor)
		query = query.Where(condition, args...)
	}

	if err := query.
		Order("is_pinned DESC, created_at DESC, id DESC").
		Limit(limit).
		Offset(offset).
		Find(&posts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch posts"})
		return
	}

	// One extra query for the whole page: which of these posts the caller has
	// already reported, so the button renders settled rather than lying.
	postIDs := make([]uuid.UUID, 0, len(posts))
	for _, post := range posts {
		postIDs = append(postIDs, post.ID)
	}
	reported := reportedPostIDs(userObj.ID, postIDs)

	results := make([]gin.H, 0, len(posts))
	for _, post := range posts {
		row := forumPostResponse(post, false)
		row["reportedByMe"] = reported[post.ID.String()]
		row["isMine"] = post.AuthorID == userObj.ID
		results = append(results, row)
	}

	// Only offer a next page when this one was full; an empty cursor is the
	// client's signal that the feed is exhausted.
	var nextCursor string
	if len(posts) == limit {
		last := posts[len(posts)-1]
		nextCursor = encodeFeedCursor(feedCursor{
			Pinned:    last.IsPinned,
			CreatedAt: last.CreatedAt,
			ID:        last.ID,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"posts":      results,
		"scope":      scope,
		"limit":      limit,
		"offset":     offset,
		"nextCursor": nextCursor,
	})
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

	// Access: geography is the boundary (decision D-014). A unit-tagged post
	// stays visible to its own unit; super admins see everything.
	lat, lng := parseCoords(c)
	if !postVisibleTo(userObj, post, lat, lng) {
		c.JSON(http.StatusForbidden, gin.H{"error": "This post is outside your area and is not open to you"})
		return
	}

	// Count the view atomically. The previous `post.ViewCount++` followed by a
	// full-row `Save` was a read-modify-write lost update (and rewrote every
	// column on a hot read path) — the same defect class as RES-004's RSVP bug.
	if err := config.DB.Model(&models.ForumPost{}).
		Where("id = ?", post.ID).
		UpdateColumn("view_count", gorm.Expr("view_count + 1")).Error; err == nil {
		post.ViewCount++
	}

	row := forumPostResponse(post, true)
	row["reportedByMe"] = reportedPostIDs(userObj.ID, []uuid.UUID{post.ID})[post.ID.String()]
	row["isMine"] = post.AuthorID == userObj.ID

	c.JSON(http.StatusOK, gin.H{
		"post": row,
	})
}

// CreateForumReply creates a reply to a forum post
func CreateForumReply(c *gin.Context) {
	var input struct {
		PostID    string   `json:"postId" binding:"required"`
		Content   string   `json:"content" binding:"required"`
		Latitude  *float64 `json:"latitude"`
		Longitude *float64 `json:"longitude"`
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
	if err := config.DB.First(&post, "id = ? AND status = ?", postID, "published").Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
		return
	}

	if post.IsLocked {
		c.JSON(http.StatusForbidden, gin.H{"error": "This post is locked"})
		return
	}

	// Visibility and participation are the same boundary (decision D-014): you
	// cannot reply to a post you could not open, so a Lagos resident cannot join
	// a Benue village thread. This is what keeps the feed from becoming noise.
	if !postVisibleTo(userObj, post, input.Latitude, input.Longitude) {
		c.JSON(http.StatusForbidden, gin.H{"error": "This discussion is outside your area"})
		return
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

	// Atomic reply count increment (no lost-update race)
	config.DB.Model(&models.ForumPost{}).Where("id = ?", postID).Update("reply_count", gorm.Expr("reply_count + 1"))

	// Atomic last reply at set
	config.DB.Model(&models.ForumPost{}).Where("id = ?", postID).Update("last_reply_at", time.Now())

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
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid unit ID"})
			return
		}
		{
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
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid expiry date"})
			return
		}
		expiresAt = &parsed
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

	// Resolve viewer place for geo scoping
	place := resolveViewerPlace(userObj, nil, nil)
	maxRank, _ := scopeRank("lga") // default scope

	// Build query with geo scope filter
	query := config.DB.Where("status = ?", "published")
	var announcements []models.CommunityAnnouncement
	
	// Super admins bypass geo filtering
	if userObj.Role != "super_admin" {
		// Apply scope window condition
		condition, args := scopeWindowCondition(place, maxRank)
		query = query.Where(condition, args...)
	}

	// Filter by role (fallback for those without location)
	if userObj.Role == "citizen" {
		if userObj.UnitID != nil {
			// For citizens without location, still filter by unit as fallback
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
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid unit ID"})
			return
		}
		{
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
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end date"})
			return
		}
		endDate = &parsed
	}

	if endDate != nil && !endDate.After(eventDate) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Event end date must be after start date"})
		return
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
	// Resolve viewer place for geo scoping
	place := resolveViewerPlace(userObj, nil, nil)
	maxRank, _ := scopeRank("lga") // default scope

	var events []models.CommunityEvent
	query := config.DB.Where("status = ?", "upcoming")

	// Super admins bypass geo filtering
	if userObj.Role != "super_admin" {
		// Apply scope window condition
		condition, args := scopeWindowCondition(place, maxRank)
		query = query.Where(condition, args...)
	}

	// Filter by role (fallback for those without location)
	if userObj.Role == "citizen" {
		if userObj.UnitID != nil {
			// For citizens without location, still filter by unit as fallback
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

	// Distinguish the failure modes inside the transaction from the HTTP
	// mapping outside it, so a validation failure never surfaces as a 500.
	var (
		errEventNotFound = errors.New("event not found")
		errAlreadyRSVPed = errors.New("already rsvped")
		errEventFull     = errors.New("event full")
	)

	var event models.CommunityEvent

	txErr := config.DB.Transaction(func(tx *gorm.DB) error {
		// Lock the event row first. This serialises every concurrent RSVP for
		// the same event, turning the old check-then-act (capacity check then
		// insert, RES-004) into a single atomic section. Without it two
		// callers can both observe a free seat and both insert.
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&event, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errEventNotFound
			}
			return err
		}

		// Derive the live attendee count instead of trusting the denormalised
		// counter. Besides being race-free under the row lock above, this
		// self-heals any historic drift of CommunityEvent.AttendeeCount.
		var current int64
		if err := tx.Model(&models.EventAttendee{}).
			Where("event_id = ?", id).
			Count(&current).Error; err != nil {
			return err
		}

		// Reject a duplicate RSVP. The partial unique index
		// idx_event_attendee_active is the database-level backstop.
		var existing models.EventAttendee
		err := tx.Where("event_id = ? AND user_id = ?", id, userObj.ID).
			First(&existing).Error
		if err == nil {
			return errAlreadyRSVPed
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if event.MaxAttendees > 0 && current >= int64(event.MaxAttendees) {
			return errEventFull
		}

		attendee := models.EventAttendee{
			EventID: id,
			UserID:  userObj.ID,
			Status:  "registered",
		}
		if err := tx.Create(&attendee).Error; err != nil {
			return err
		}

		// Keep the denormalised counter equal to the rows it summarises.
		if err := tx.Model(&models.CommunityEvent{}).
			Where("id = ?", id).
			Update("attendee_count", current+1).Error; err != nil {
			return err
		}
		return nil
	})

	switch {
	case txErr == nil:
		c.JSON(http.StatusOK, gin.H{"message": "RSVP successful"})
	case errors.Is(txErr, errEventNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
	case errors.Is(txErr, errAlreadyRSVPed):
		c.JSON(http.StatusBadRequest, gin.H{"error": "You have already RSVP'd to this event"})
	case errors.Is(txErr, errEventFull):
		c.JSON(http.StatusBadRequest, gin.H{"error": "Event is full"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to RSVP"})
	}
}
