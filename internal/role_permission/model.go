package rolepermission

import (
	model "sound-stage-backend/internal/model"
)

type RolePermission struct {
	model.BaseModel
	RoleID       uint `gorm:"not null" json:"roleID"`
	PermissionID uint `gorm:"not null" json:"permissionID"`
}

func (RolePermission) TableName() string {
	return "role_permissions"
}
