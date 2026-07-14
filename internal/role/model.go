package role

import (
	model "sound-stage-backend/internal/model"
)

const (
	RoleAdmin     = "admin"
	RoleListener  = "listener"
	RoleSpeaker   = "speaker"
	RoleModerator = "moderator"
)

type Role struct {
	model.BaseModel
	Name        string  `gorm:"not null" json:"name" validate:"required"`
	Description *string `json:"description,omitempty"`
}

func (Role) TableName() string {
	return "roles"
}
