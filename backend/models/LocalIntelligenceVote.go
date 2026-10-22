package models

import (
	"time"

	"github.com/google/uuid"
)

// LocalIntelligenceVote records one upvote per user per fact. The unique
// composite index makes concurrent repeat taps idempotent at the database
// level, so the handler can insert-on-conflict instead of read-then-write.
type LocalIntelligenceVote struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	IntelID   uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_local_intel_vote" json:"intelId"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_local_intel_vote" json:"userId"`
	CreatedAt time.Time `json:"createdAt"`
}

func (LocalIntelligenceVote) TableName() string {
	return "local_intel_votes"
}
