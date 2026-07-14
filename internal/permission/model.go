package permission

import (
	model "sound-stage-backend/internal/model"
)

type Permission struct {
	model.BaseModel
	Name     string `gorm:"not null;uniqueIndex" json:"name" validate:"required"`
	Resource string `gorm:"not null" json:"resource" validate:"required"`
	Action   string `gorm:"not null" json:"action" validate:"required"`
}

func (Permission) TableName() string {
	return "permissions"
}
