package room

import (
	"sound-stage-backend/internal/category"
	model "sound-stage-backend/internal/model"
	"sound-stage-backend/internal/pkg/listopts"
	"sound-stage-backend/internal/role"
	"sound-stage-backend/internal/tag"
	user "sound-stage-backend/internal/user"
	"strings"

	"gorm.io/gorm"
)

type RoomType string

const (
	RoomTypePublic  RoomType = "public"
	RoomTypePrivate RoomType = "private"
)

type Room struct {
	model.BaseModel
	Name        string              `gorm:"not null" json:"name" validate:"required"`
	Description string              `json:"description" validate:"required"`
	CreatorID   uint                `json:"creatorID" validate:"required"`
	Creator     user.User           `gorm:"foreignKey:CreatorID"`
	Users       []user.User         `gorm:"many2many:room_users"`
	Categories  []category.Category `gorm:"many2many:room_categories"`
	Tags        []tag.Tag           `gorm:"-"`
	Type        RoomType            `gorm:"default:public" validate:"required,oneof=public private"`
	PrivateCode *string             `validate:"omitempty"`
	DeletedAt   gorm.DeletedAt
}

func (Room) TableName() string {
	return "rooms"
}

type CreateRoomParams struct {
	Name        string   `json:"name" validate:"required"`
	Description string   `json:"description"`
	CreatorID   uint     `json:"creatorID" validate:"required"`
	CategoryIds []uint   `json:"categoryIds" validate:"lte=3"`
	TagIds      []uint   `json:"tagIds" validate:"lte=5"`
	Type        RoomType `validate:"required"`
	privateCode *string
}

type UpdateUserRoleParams struct {
	Role role.RoleName `json:"role" validate:"required"`
}

type UpdateRoomParams struct {
	Name        string   `json:"name" validate:"required"`
	Description string   `json:"description" validate:"omitempty"`
	CategoryIds []uint   `json:"categoryIds" validate:"lte=3"`
	TagIds      []uint   `json:"tagIds" validate:"lte=5"`
	Type        RoomType `validate:"required"`
	privateCode *string
}

type RoomFilter struct {
	Query string `form:"query"`
}

var allowedSortFields = map[string]string{
	"name":       "rooms.name",
	"created_at": "rooms.created_at",
	"updated_at": "rooms.updated_at",
}

func FilterBySearch(query string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if query == "" {
			return db
		}
		return db.Where("LOWER(rooms.name) LIKE LOWER(?)", "%"+query+"%")
	}
}

func Filters(f RoomFilter) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Scopes(
			FilterBySearch(f.Query),
		)
	}
}

func Sort(s listopts.Sort) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		col, ok := allowedSortFields[strings.ToLower(s.Field)]
		if !ok {
			col = "rooms.created_at"
		}

		order := strings.ToLower(s.Order)
		if order != "asc" && order != "desc" {
			order = "desc"
		}
		return db.Order(col + " " + order)
	}
}
