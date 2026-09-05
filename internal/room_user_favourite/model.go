package roomuserfavourite

import (
	model "sound-stage-backend/internal/model"
	"sound-stage-backend/internal/room"
)

type RoomUserFavourite struct {
	model.BaseModel
	UserID uint      `gorm:"uniqueIndex:idx_room_user_favourites_user_room"`
	RoomID uint      `gorm:"uniqueIndex:idx_room_user_favourites_user_room"`
	Room   room.Room `gorm:"foreignKey:RoomID"`
}

func (RoomUserFavourite) TableName() string {
	return "room_user_favourites"
}
