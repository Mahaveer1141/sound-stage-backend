package roomcategory

import (
	model "sound-stage-backend/internal/model"
)

type RoomCategory struct {
	model.BaseModel
	RoomID     uint `gorm:"not null;uniqueIndex:idx_room_category" json:"roomId"`
	CategoryID uint `gorm:"not null;uniqueIndex:idx_room_category" json:"categoryId"`
}

func (RoomCategory) TableName() string {
	return "room_categories"
}
