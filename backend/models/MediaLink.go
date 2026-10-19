package models

import (
	"time"

	"github.com/google/uuid"
)

// MediaLink is one attachment reference.
//
// The object store is content-addressed: identical bytes deduplicate to a single
// key (services/file_storage_service.go, services/r2_service.go). An attachment
// is therefore a *link*, not ownership — several parents can point at the same
// object, so the object may only be removed once the last link to it is gone.
// The avatar/cover "delete the old file on replace" shortcut is unsafe here
// (docs/community/MEDIA-ATTACHMENTS-PLAN.md §5.3); this table is the refcount.
//
// The unique index makes a duplicate attach idempotent rather than an error.
type MediaLink struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ObjectKey  string    `gorm:"type:varchar(255);not null;uniqueIndex:idx_media_link_unique,priority:1;index:idx_media_link_object" json:"objectKey"`
	ParentType string    `gorm:"type:varchar(24);not null;uniqueIndex:idx_media_link_unique,priority:2" json:"parentType"`
	ParentID   uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_media_link_unique,priority:3" json:"parentId"`
	CreatedAt  time.Time `json:"createdAt"`
}

func (MediaLink) TableName() string {
	return "media_links"
}
