package chatmessage

import (
	"strings"

	"sound-stage-backend/internal/model"
	"sound-stage-backend/internal/pkg/listopts"
	"sound-stage-backend/internal/user"

	"gorm.io/gorm"
)

type ChatMessage struct {
	model.BaseModel
	RoomID   uint      `gorm:"not null"`
	UserID   uint      `gorm:"not null"`
	Content  string    `gorm:"not null"`
	IsPinned bool      `gorm:"not null;default:false"`
	User     user.User `gorm:"foreignKey:UserID"`
}

func (ChatMessage) TableName() string {
	return "chat_messages"
}

const MaxPinnedMessages = 20

type CreateChatMessageParams struct {
	RoomID   uint   `json:"roomID" validate:"required"`
	UserID   uint   `json:"userID" validate:"required"`
	Content  string `json:"content" validate:"required,max=2000"`
	IsPinned bool   `json:"isPinned"`
}

type ChatMessageFilter struct {
	RoomID   uint  `form:"roomId"`
	IsPinned *bool `form:"isPinned"`
}

type ChatMessageResponse struct {
	ID        uint               `json:"id"`
	RoomID    uint               `json:"roomID"`
	Content   string             `json:"content"`
	IsPinned  bool               `json:"isPinned"`
	CreatedAt string             `json:"createdAt"`
	UpdatedAt string             `json:"updatedAt"`
	User      *user.UserResponse `json:"user,omitempty"`
}

var allowedSortFields = map[string]string{
	"created_at": "chat_messages.created_at",
}

func FilterByRoom(roomID uint) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if roomID == 0 {
			return db
		}
		return db.Where("chat_messages.room_id = ?", roomID)
	}
}

func FilterByIsPinned(isPinned *bool) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if isPinned == nil {
			return db
		}
		return db.Where("chat_messages.is_pinned = ?", *isPinned)
	}
}

func Filters(f ChatMessageFilter) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Scopes(
			FilterByRoom(f.RoomID),
			FilterByIsPinned(f.IsPinned),
		)
	}
}

func Sort(s listopts.Sort) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		col, ok := allowedSortFields[strings.ToLower(s.Field)]
		if !ok {
			col = "chat_messages.created_at"
		}

		order := strings.ToLower(s.Order)
		if order != "asc" && order != "desc" {
			order = "desc"
		}
		return db.Order(col + " " + order)
	}
}
