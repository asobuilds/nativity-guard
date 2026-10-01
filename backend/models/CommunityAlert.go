package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CommunityAlert represents an alert sent to the community.
type CommunityAlert struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Title       string         `gorm:"not null" json:"title"`
	Content     string         `gorm:"type:text;not null" json:"content"`
	Type        string         `gorm:"not null" json:"type"`           // security, weather, health, community, emergency
	Severity    string         `gorm:"default:medium" json:"severity"` // low, medium, high, critical
	Location    string         `json:"location"`
	Latitude    float64        `json:"latitude"`
	Longitude   float64        `json:"longitude"`
	Radius      float64        `json:"radius"`                       // Coverage radius in km
	Status      string         `gorm:"default:active" json:"status"` // active, expired, resolved
	ExpiresAt   *time.Time     `json:"expiresAt"`
	CreatedBy   uuid.UUID      `gorm:"type:uuid;not null" json:"createdBy"`
	ConfirmedBy *uuid.UUID     `gorm:"type:uuid" json:"confirmedBy,omitempty"`
	ConfirmedAt *time.Time     `json:"confirmedAt,omitempty"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	Author    User `gorm:"foreignKey:CreatedBy" json:"author,omitempty"`
	Confirmer User `gorm:"foreignKey:ConfirmedBy" json:"confirmer,omitempty"`
}

func (CommunityAlert) TableName() string {
	return "community_alerts"
}

// AlertSubscription tracks users subscribed to alerts.
//
// Latitude and Longitude are where the subscriber wants to receive nearby
// alerts from. Both zero means "no location set" — the subscriber then
// receives every alert in their chosen categories, nationwide.
type AlertSubscription struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID      `gorm:"type:uuid;not null" json:"userId"`
	UnitID    *uuid.UUID     `gorm:"type:uuid" json:"unitId,omitempty"`
	Type      string         `gorm:"not null" json:"type"`          // CSV: security,weather,community,general or "all"
	Channel   string         `gorm:"default:in_app" json:"channel"` // CSV: in_app,email,sms
	Location  string         `json:"location"`
	Latitude  float64        `json:"latitude"`
	Longitude float64        `json:"longitude"`
	Radius    float64        `json:"radius"` // Notification radius in km
	IsActive  bool           `gorm:"default:true" json:"isActive"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	User User          `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Unit *SecurityUnit `gorm:"foreignKey:UnitID" json:"unit,omitempty"`
}

func (AlertSubscription) TableName() string {
	return "alert_subscriptions"
}