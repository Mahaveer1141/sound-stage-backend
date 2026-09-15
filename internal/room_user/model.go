package roomuser

import (
	model "sound-stage-backend/internal/model"
	"sound-stage-backend/internal/pkg/listopts"
	"sound-stage-backend/internal/role"
	user "sound-stage-backend/internal/user"
	"strings"

	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RoomUser struct {
	model.BaseModel
	UserID       uint
	RoomID       uint
	RoleID       uint
	User         user.User `gorm:"foreignKey:UserID"`
	Role         role.Role `gorm:"foreignKey:RoleID"`
	LastJoinedAt time.Time
	LastLeftAt   time.Time
	IsOnline     bool
	IsBlocked    bool
	BlockedByID  *uint
	BlockedBy    *user.User `gorm:"foreignKey:BlockedByID"`
	IsMuted      bool       `gorm:"-"`
	IsHandRaised bool       `gorm:"-"`
}

func (RoomUser) TableName() string {
	return "room_users"
}

type RoomUserFilter struct {
	Roles    []string `form:"roles"`
	IsOnline *bool    `form:"isOnline"`
	Query    string   `form:"query"`
}

type RoomUserResponse struct {
	ID           uint              `json:"id"`
	User         user.UserResponse `json:"user"`
	Role         role.RoleResponse `json:"role"`
	LastJoinedAt string            `json:"lastJoinedAt"`
	LastLeftAt   string            `json:"lastLeftAt"`
	IsOnline     bool              `json:"isOnline"`
	IsMuted      bool              `json:"isMuted"`
	IsHandRaised bool              `json:"isHandRaised"`
	CanManage    bool              `json:"canManage"`
	CanSpeak     bool              `json:"canSpeak"`
	IsAdmin      bool              `json:"isAdmin"`
	IsOwner      bool              `json:"isOwner"`
}

var allowedUserSortFields = map[string]string{
	"created_at":     "room_users.created_at",
	"id":             "room_users.id",
	"last_joined_at": "room_users.last_joined_at",
}

func (ru *RoomUser) IsListener() bool {
	return ru.Role.Name == role.RoleListener
}

func (ru *RoomUser) IsAdmin() bool {
	return ru.IsOwner() || ru.Role.Name == role.RoleAdmin
}

func (ru *RoomUser) IsOwner() bool {
	return ru.Role.Name == role.RoleOwner
}

func (ru *RoomUser) CanManage() bool {
	return ru.IsAdmin() || ru.Role.Name == role.RoleModerator
}

func (ru *RoomUser) CanSpeak() bool {
	return ru.CanManage() || ru.Role.Name == role.RoleSpeaker
}

func FilterByRoles(roles []string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if len(roles) > 0 {
			db = db.Joins("JOIN roles ON roles.id = room_users.role_id").Where("roles.name IN (?)", roles)
		}
		return db
	}
}

func FilterByIsOnline(isOnline *bool) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if isOnline != nil {
			return db.Where("is_online = ?", *isOnline)
		}
		return db
	}
}

func FilterByQuery(query string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		query = strings.TrimSpace(query)
		if query == "" {
			return db
		}
		pattern := "%" + query + "%"
		return db.
			Joins("JOIN users ON users.id = room_users.user_id").
			Where("(users.first_name || ' ' || COALESCE(users.last_name, '')) ILIKE ?", pattern)
	}
}

func Filters(f RoomUserFilter) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Scopes(
			FilterByRoles(f.Roles),
			FilterByIsOnline(f.IsOnline),
			FilterByQuery(f.Query),
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
		return db.Order(col + " " + order).Order("room_users.id " + order)
	}
}

func SortByUserIDs(userIDs []uint) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(userIDs)), ",")
		args := make([]any, len(userIDs))
		for i, id := range userIDs {
			args[i] = id
		}
		return db.Clauses(clause.OrderBy{
			Expression: clause.Expr{SQL: "array_position(ARRAY[" + placeholders + "]::bigint[], user_id)", Vars: args},
		})
	}
}
