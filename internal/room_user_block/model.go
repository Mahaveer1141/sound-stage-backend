package roomuserblock

import (
	model "sound-stage-backend/internal/model"
)

type RoomUserBlock struct {
	model.BaseModel
	UserID      uint `gorm:"uniqueIndex:idx_room_user_blocks_room_user"`
	RoomID      uint `gorm:"uniqueIndex:idx_room_user_blocks_room_user"`
	BlockedByID uint
}

func (RoomUserBlock) TableName() string {
	return "room_user_blocks"
}
