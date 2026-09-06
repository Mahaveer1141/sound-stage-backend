package room

import (
	"mime/multipart"
	"sound-stage-backend/internal/category"
	fileattachment "sound-stage-backend/internal/file_attachment"
	model "sound-stage-backend/internal/model"
	"sound-stage-backend/internal/pkg/httpx"
	"sound-stage-backend/internal/pkg/listopts"
	roomuser "sound-stage-backend/internal/room_user"
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
	Name        string                         `gorm:"not null" validate:"required"`
	Description string                         `validate:"required"`
	CreatorID   uint                           `validate:"required"`
	Creator     user.User                      `gorm:"foreignKey:CreatorID"`
	Users       []user.User                    `gorm:"many2many:room_users"`
	Categories  []category.Category            `gorm:"many2many:room_categories"`
	Tags        []tag.Tag                      `gorm:"-"`
	CoverImage  *fileattachment.FileAttachment `gorm:"polymorphic:Owner;"`
	LogoImage   *fileattachment.FileAttachment `gorm:"polymorphic:Owner;"`
	Type        RoomType                       `gorm:"default:public" validate:"required,oneof=public private"`
	PrivateCode *string                        `validate:"omitempty"`
	DeletedAt   gorm.DeletedAt
}

func (Room) TableName() string {
	return "rooms"
}

type CreateRoomParams struct {
	Name        string                `json:"name" form:"name" validate:"required"`
	Description string                `json:"description" form:"description"`
	CreatorID   uint                  `json:"creatorID" validate:"required"`
	CoverImage  *multipart.FileHeader `json:"-" form:"coverImage" validate:"omitempty,max_size=10485760"`
	LogoImage   *multipart.FileHeader `json:"-" form:"logoImage" validate:"omitempty,max_size=10485760"`
	CategoryIds []uint                `json:"categoryIds" form:"categoryIds" validate:"lte=3"`
	TagIds      []uint                `json:"tagIds" form:"tagIds" validate:"lte=5"`
	Type        RoomType              `json:"type" form:"type" validate:"required"`
	privateCode *string
}

type UpdateRoomParams struct {
	Name        string                `json:"name" form:"name" validate:"required"`
	Description string                `json:"description" form:"description" validate:"omitempty"`
	CoverImage  *multipart.FileHeader `json:"-" form:"coverImage" validate:"omitempty,max_size=10485760"`
	LogoImage   *multipart.FileHeader `json:"-" form:"logoImage" validate:"omitempty,max_size=10485760"`
	CategoryIds []uint                `json:"categoryIds" form:"categoryIds" validate:"lte=3"`
	TagIds      []uint                `json:"tagIds" form:"tagIds" validate:"lte=5"`
	Type        RoomType              `json:"type" form:"type" validate:"required"`
	privateCode *string
}

type RoomResponse struct {
	ID           uint                          `json:"id"`
	Name         string                        `json:"name"`
	Description  string                        `json:"description"`
	Type         RoomType                      `json:"type"`
	PrivateCode  *string                       `json:"privateCode,omitempty"`
	CoverImage   *httpx.FileAttachmentResponse `json:"coverImage,omitempty"`
	LogoImage    *httpx.FileAttachmentResponse `json:"logoImage,omitempty"`
	Categories   []category.CategoryResponse   `json:"categories,omitempty"`
	Tags         []tag.TagResponse             `json:"tags,omitempty"`
	IsRoomUser   bool                          `json:"isRoomUser"`
	IsFavourited bool                          `json:"isFavourited"`
}

type RoomViewer struct {
	RoomUser     *roomuser.RoomUser
	IsFavourited bool
}

type RoomFilter struct {
	Query       string    `form:"query"`
	CategoryIds []uint    `form:"categoryIds"`
	TagIds      []uint    `form:"tagIds"`
	Type        *RoomType `form:"type"`
	UserID      uint      `form:"-"`
}

var allowedSortFields = map[string]string{
	"name":       "rooms.name",
	"created_at": "rooms.created_at",
	"updated_at": "rooms.updated_at",
}

func FilterBySearch(query string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		query = strings.TrimSpace(query)
		if query == "" {
			return db
		}
		return db.Where("rooms.name LIKE ?", "%"+query+"%")
	}
}

func FilterByCategories(categoryIds []uint) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if len(categoryIds) > 0 {
			db = db.Joins("JOIN room_categories ON room_categories.room_id = rooms.id").
				Where("room_categories.category_id IN (?)", categoryIds)
		}
		return db
	}
}

func FilterByTags(tagIds []uint) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if len(tagIds) > 0 {
			db = db.Joins("JOIN taggables ON taggables.taggable_type = ? AND taggables.taggable_id = rooms.id",
				"rooms", tagIds).
				Where("taggables.tag_id IN (?)", tagIds)
		}
		return db
	}
}

func FilterByType(roomType *RoomType) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if roomType != nil && strings.TrimSpace(string(*roomType)) != "" {
			db = db.Where("type = ?", roomType)
		}
		return db
	}
}

func FilterByNotBlocked(userID uint) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if userID == 0 {
			return db
		}
		return db.Where(
			"NOT EXISTS (SELECT 1 FROM room_users WHERE room_users.room_id = rooms.id AND room_users.user_id = ? AND room_users.is_blocked = ?)",
			userID, true,
		)
	}
}

func Filters(f RoomFilter) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Scopes(
			FilterBySearch(f.Query),
			FilterByCategories(f.CategoryIds),
			FilterByTags(f.TagIds),
			FilterByType(f.Type),
			FilterByNotBlocked(f.UserID),
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
