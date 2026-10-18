package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

// GeoPlace is the server-derived geographic address of community content.
//
// Names are never taken from the request (decision D-012): the client sends a
// scope and, optionally, coordinates; the server resolves the place from the
// author's verified unit or from the cached reverse geocoder. LGA names repeat
// across states, so an LGA is only ever meaningful together with its state.
//
// Scope is one of: locality, lga, state, country, world (D-011). Rows written
// before geography existed carry the column default "world", so a post of
// unknown place is never presented as local.
type GeoPlace struct {
	Scope     string  `gorm:"type:varchar(16);not null;default:'world'" json:"scope"`
	Locality  string  `gorm:"type:varchar(160)" json:"locality,omitempty"`
	LGA       string  `gorm:"type:varchar(160)" json:"lga,omitempty"`
	State     string  `gorm:"type:varchar(160)" json:"state,omitempty"`
	Country   string  `gorm:"type:varchar(80);default:'Nigeria'" json:"country,omitempty"`
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
	Geohash   string  `gorm:"type:varchar(12)" json:"geohash,omitempty"`
}

// ForumPost represents a community forum post
type ForumPost struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UnitID      *uuid.UUID     `gorm:"type:uuid" json:"unitId,omitempty"`
	GeoPlace    GeoPlace       `gorm:"embedded" json:"place"`
	Title       string         `gorm:"not null" json:"title"`
	Content     string         `gorm:"type:text;not null" json:"content"`
	Category    string         `gorm:"default:general" json:"category"`
	AuthorID    uuid.UUID      `gorm:"type:uuid;not null" json:"authorId"`
	Status      string         `gorm:"default:published" json:"status"`
	IsPinned    bool           `gorm:"default:false" json:"isPinned"`
	IsLocked    bool           `gorm:"default:false" json:"isLocked"`
	ViewCount   int            `gorm:"default:0" json:"viewCount"`
	ReplyCount  int            `gorm:"default:0" json:"replyCount"`
	LastReplyAt *time.Time     `json:"lastReplyAt"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	Author  User          `gorm:"foreignKey:AuthorID" json:"author,omitempty"`
	Unit    *SecurityUnit `gorm:"foreignKey:UnitID" json:"unit,omitempty"`
	Replies []ForumReply  `gorm:"foreignKey:PostID" json:"replies,omitempty"`
}

func (ForumPost) TableName() string {
	return "forum_posts"
}

// ForumReply represents a reply to a forum post
type ForumReply struct {
	ID         uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	PostID     uuid.UUID      `gorm:"type:uuid;not null" json:"postId"`
	Content    string         `gorm:"type:text;not null" json:"content"`
	AuthorID   uuid.UUID      `gorm:"type:uuid;not null" json:"authorId"`
	IsSolution bool           `gorm:"default:false" json:"isSolution"`
	Status     string         `gorm:"default:published" json:"status"`
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`

	Post   ForumPost `gorm:"foreignKey:PostID" json:"post,omitempty"`
	Author User      `gorm:"foreignKey:AuthorID" json:"author,omitempty"`
}

func (ForumReply) TableName() string {
	return "forum_replies"
}

// CommunityAnnouncement represents community announcements
type CommunityAnnouncement struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UnitID      *uuid.UUID     `gorm:"type:uuid" json:"unitId,omitempty"`
	Title       string         `gorm:"not null" json:"title"`
	Content     string         `gorm:"type:text;not null" json:"content"`
	Type        string         `gorm:"default:general" json:"type"`
	Priority    string         `gorm:"default:normal" json:"priority"`
	AuthorID    uuid.UUID      `gorm:"type:uuid;not null" json:"authorId"`
	Status      string         `gorm:"default:published" json:"status"`
	ExpiresAt   *time.Time     `json:"expiresAt"`
	PublishedAt time.Time      `json:"publishedAt"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	Author User          `gorm:"foreignKey:AuthorID" json:"author,omitempty"`
	Unit   *SecurityUnit `gorm:"foreignKey:UnitID" json:"unit,omitempty"`
}

func (CommunityAnnouncement) TableName() string {
	return "community_announcements"
}

// CommunityEvent represents community events
type CommunityEvent struct {
	ID            uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UnitID        *uuid.UUID     `gorm:"type:uuid" json:"unitId,omitempty"`
	Title         string         `gorm:"not null" json:"title"`
	Description   string         `gorm:"type:text;not null" json:"description"`
	Location      string         `json:"location"`
	Latitude      float64        `json:"latitude"`
	Longitude     float64        `json:"longitude"`
	EventDate     time.Time      `json:"eventDate"`
	EndDate       *time.Time     `json:"endDate"`
	Type          string         `gorm:"default:meeting" json:"type"`
	Status        string         `gorm:"default:upcoming" json:"status"`
	MaxAttendees  int            `gorm:"default:0" json:"maxAttendees"`
	AttendeeCount int            `gorm:"default:0" json:"attendeeCount"`
	AuthorID      uuid.UUID      `gorm:"type:uuid;not null" json:"authorId"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`

	Author User          `gorm:"foreignKey:AuthorID" json:"author,omitempty"`
	Unit   *SecurityUnit `gorm:"foreignKey:UnitID" json:"unit,omitempty"`
}

func (CommunityEvent) TableName() string {
	return "community_events"
}

// EventAttendee tracks event attendees.
//
// A user may hold at most one live RSVP per event. Because EventAttendee is
// soft-deleted, that uniqueness is a *partial* index over
// (event_id, user_id) WHERE deleted_at IS NULL — otherwise a cancelled RSVP
// would permanently block re-registration. AutoMigrate creates it; a duplicate
// live row therefore fails the migration loudly and must be resolved first.
type EventAttendee struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	EventID   uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex:idx_event_attendee_active,where:deleted_at IS NULL" json:"eventId"`
	UserID    uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex:idx_event_attendee_active,where:deleted_at IS NULL" json:"userId"`
	Status    string         `gorm:"default:registered" json:"status"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Event CommunityEvent `gorm:"foreignKey:EventID" json:"event,omitempty"`
	User  User           `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (EventAttendee) TableName() string {
	return "event_attendees"
}
