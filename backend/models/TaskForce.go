package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	TaskForceStatusActive    = "active"
	TaskForceStatusDisbanded = "disbanded"

	TaskForceMemberRoleHost   = "host"
	TaskForceMemberRoleMember = "member"

	TaskForceMemberStatusInvited = "invited"
	TaskForceMemberStatusActive  = "active"
	TaskForceMemberStatusLeft    = "left"
	TaskForceMemberStatusRemoved = "removed"

	TaskForceOfficerStatusActive  = "active"
	TaskForceOfficerStatusRemoved = "removed"

	TaskForceInviteStatusPending  = "pending"
	TaskForceInviteStatusAccepted = "accepted"
	TaskForceInviteStatusRevoked  = "revoked"
	TaskForceInviteStatusExpired  = "expired"

	TaskForceRoomType = "task_force"
)

type TaskForce struct {
	ID         uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name       string         `gorm:"not null" json:"name"`
	HostUnitID uuid.UUID      `gorm:"type:uuid;not null;index" json:"hostUnitId"`
	Status     string         `gorm:"not null;default:active;index" json:"status"`
	CreatedBy  uuid.UUID      `gorm:"type:uuid;not null" json:"createdBy"`
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`

	HostUnit SecurityUnit `gorm:"foreignKey:HostUnitID" json:"hostUnit,omitempty"`
	Creator  User         `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
}

func (TaskForce) TableName() string {
	return "task_forces"
}

type TaskForceMember struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	TaskForceID uuid.UUID      `gorm:"type:uuid;not null;index" json:"taskForceId"`
	UnitID      uuid.UUID      `gorm:"type:uuid;not null;index" json:"unitId"`
	Role        string         `gorm:"not null;default:member" json:"role"`
	Status      string         `gorm:"not null;default:invited;index" json:"status"`
	AcceptedAt  *time.Time     `json:"acceptedAt,omitempty"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	TaskForce TaskForce    `gorm:"foreignKey:TaskForceID" json:"-"`
	Unit      SecurityUnit `gorm:"foreignKey:UnitID" json:"unit,omitempty"`
}

func (TaskForceMember) TableName() string {
	return "task_force_members"
}

type TaskForceOfficer struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	TaskForceID uuid.UUID      `gorm:"type:uuid;not null;index" json:"taskForceId"`
	OfficerID   uuid.UUID      `gorm:"type:uuid;not null;index" json:"officerId"`
	UnitID      uuid.UUID      `gorm:"type:uuid;not null;index" json:"unitId"`
	Status      string         `gorm:"not null;default:active;index" json:"status"`
	EnlistedAt  time.Time      `gorm:"not null;default:now()" json:"enlistedAt"`
	EnlistedBy  *uuid.UUID     `gorm:"type:uuid" json:"enlistedBy,omitempty"`
	RemovedAt   *time.Time     `json:"removedAt,omitempty"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	TaskForce TaskForce    `gorm:"foreignKey:TaskForceID" json:"-"`
	Officer   Officer      `gorm:"foreignKey:OfficerID" json:"officer,omitempty"`
	Unit      SecurityUnit `gorm:"foreignKey:UnitID" json:"unit,omitempty"`
}

func (TaskForceOfficer) TableName() string {
	return "task_force_officers"
}

type TaskForceInvite struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	TaskForceID uuid.UUID      `gorm:"type:uuid;not null;index" json:"taskForceId"`
	UnitID      uuid.UUID      `gorm:"type:uuid;not null;index" json:"unitId"`
	CodeHash    string         `gorm:"type:varchar(64);not null;index" json:"-"`
	Status      string         `gorm:"not null;default:pending;index" json:"status"`
	ExpiresAt   *time.Time     `gorm:"index" json:"expiresAt,omitempty"`
	MaxUses     int            `gorm:"not null;default:1" json:"maxUses"`
	UseCount    int            `gorm:"not null;default:0" json:"useCount"`
	CreatedBy   uuid.UUID      `gorm:"type:uuid;not null" json:"createdBy"`
	AcceptedBy  *uuid.UUID     `gorm:"type:uuid" json:"acceptedBy,omitempty"`
	AcceptedAt  *time.Time     `json:"acceptedAt,omitempty"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	TaskForce TaskForce    `gorm:"foreignKey:TaskForceID" json:"-"`
	Unit      SecurityUnit `gorm:"foreignKey:UnitID" json:"unit,omitempty"`
	Creator   User         `gorm:"foreignKey:CreatedBy" json:"-"`
}

func (TaskForceInvite) TableName() string {
	return "task_force_invites"
}

type TaskForceChannel struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	TaskForceID uuid.UUID      `gorm:"type:uuid;not null;index" json:"taskForceId"`
	RoomID      uuid.UUID      `gorm:"type:uuid;not null;index" json:"roomId"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	TaskForce TaskForce         `gorm:"foreignKey:TaskForceID" json:"-"`
	Room      CommunicationRoom `gorm:"foreignKey:RoomID" json:"room,omitempty"`
}

func (TaskForceChannel) TableName() string {
	return "task_force_channels"
}
