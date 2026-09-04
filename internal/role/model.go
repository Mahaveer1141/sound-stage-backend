package role

import (
	"slices"
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

var RoleModerationPermissions = map[RoleName][]RoleName{
	RoleOwner:     {RoleOwner, RoleAdmin, RoleModerator, RoleSpeaker, RoleListener},
	RoleAdmin:     {RoleAdmin, RoleModerator, RoleSpeaker, RoleListener},
	RoleModerator: {RoleModerator, RoleSpeaker, RoleListener},
	RoleSpeaker:   {},
	RoleListener:  {},
}

func CanModerate(actorRole, targetRole RoleName) bool {
	allowed, ok := RoleModerationPermissions[actorRole]
	if !ok {
		return false
	}
	return slices.Contains(allowed, targetRole)
}

type Role struct {
	model.BaseModel
	Name        RoleName `gorm:"not null;uniqueIndex"`
	Description *string
}

type RoleResponse struct {
	ID          uint    `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

func (Role) TableName() string {
	return "roles"
}
