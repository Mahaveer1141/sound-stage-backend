package roomuserblock

import (
	model "sound-stage-backend/internal/model"
	"sound-stage-backend/internal/user"
)

type RoomUserBlock struct {
	model.BaseModel
	UserID      uint      `gorm:"uniqueIndex:idx_room_user_blocks_room_user"`
	RoomID      uint      `gorm:"uniqueIndex:idx_room_user_blocks_room_user"`
	BlockedByID uint      `gorm:"not null"`
	User        user.User `gorm:"foreignKey:UserID"`
	BlockedBy   user.User `gorm:"foreignKey:BlockedByID"`
}

func (RoomUserBlock) TableName() string {
	return "room_user_blocks"
}

type RoomUserBlockResponse struct {
	ID        uint              `json:"id"`
	User      user.UserResponse `json:"user"`
	BlockedBy user.UserResponse `json:"blockedBy"`
	CreatedAt string            `json:"createdAt"`
}
