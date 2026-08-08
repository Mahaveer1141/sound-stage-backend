package roomuser

import (
	model "sound-stage-backend/internal/model"
	"sound-stage-backend/internal/pkg/listopts"
	"sound-stage-backend/internal/role"
	user "sound-stage-backend/internal/user"
	"strings"

	"time"

	"gorm.io/gorm"
)

type RoomUser struct {
	model.BaseModel
	UserID       uint
	RoomID       uint
	RoleID       uint
	User         user.User `gorm:"foreignKey:UserID" json:"user"`
	Role         role.Role `gorm:"foreignKey:RoleID" json:"role"`
	LastJoinedAt time.Time `json:"lastJoinedAt"`
	LastLeftAt   time.Time `json:"lastLeftAt"`
	IsOnline     bool      `json:"isOnline"`
}

func (RoomUser) TableName() string {
	return "room_users"
}

type RoomUserFilter struct {
	Roles []string `form:"roles"`
}

var allowedUserSortFields = map[string]string{
	"created_at": "room_users.created_at",
}

func (ru *RoomUser) IsListener() bool {
	return ru.Role.Name == string(role.RoleListener)
}

func FilterByRoles(roles []string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if len(roles) > 0 {
			db = db.Joins("JOIN roles ON roles.id = room_users.role_id").Where("roles.name IN (?)", roles)
		}
		return db
	}
}

func Filters(f RoomUserFilter) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Scopes(
			FilterByRoles(f.Roles),
		)
	}
}

func Sort(s listopts.Sort) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		col, ok := allowedUserSortFields[strings.ToLower(s.Field)]
		if !ok {
			col = "room_users.created_at"
		}
		order := strings.ToLower(s.Order)
		if order != "asc" && order != "desc" {
			order = "desc"
		}
		return db.Order(col + " " + order)
	}
}
