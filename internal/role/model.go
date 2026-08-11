package role

import (
	model "sound-stage-backend/internal/model"
)

type RoleName string

const (
	RoleOwner     RoleName = "owner"
	RoleAdmin     RoleName = "admin"
	RoleListener  RoleName = "listener"
	RoleSpeaker   RoleName = "speaker"
	RoleModerator RoleName = "moderator"
)

var RoleAssignmentPermissions = map[RoleName][]RoleName{
	RoleOwner:     {},
	RoleListener:  {RoleOwner, RoleAdmin, RoleModerator},
	RoleSpeaker:   {RoleOwner, RoleAdmin, RoleModerator},
	RoleModerator: {RoleOwner, RoleAdmin},
	RoleAdmin:     {RoleOwner, RoleAdmin},
}

type Role struct {
	model.BaseModel
	Name        string  `gorm:"not null" json:"name" validate:"required"`
	Description *string `json:"description,omitempty"`
}

func (Role) TableName() string {
	return "roles"
}
