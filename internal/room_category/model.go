package roomcategory

import (
	model "sound-stage-backend/internal/model"
)

type RoomCategory struct {
	model.BaseModel
	RoomID     uint `gorm:"not null;uniqueIndex:idx_room_category"`
	CategoryID uint `gorm:"not null;uniqueIndex:idx_room_category"`
}

func (RoomCategory) TableName() string {
	return "room_categories"
}
