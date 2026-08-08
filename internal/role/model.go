package role

import (
	model "sound-stage-backend/internal/model"
)

type RoleName string

const (
	RoleAdmin     RoleName = "admin"
	RoleListener  RoleName = "listener"
	RoleSpeaker   RoleName = "speaker"
	RoleModerator RoleName = "moderator"
)

var RoleAssignmentPermissions = map[RoleName][]RoleName{
	RoleListener:  {RoleAdmin, RoleModerator},
	RoleSpeaker:   {RoleAdmin, RoleModerator},
	RoleModerator: {RoleAdmin},
	RoleAdmin:     {RoleAdmin},
}

type Role struct {
	model.BaseModel
	Name        string  `gorm:"not null" json:"name" validate:"required"`
	Description *string `json:"description,omitempty"`
}

func (Role) TableName() string {
	return "roles"
}
