package room

import (
	model "sound-stage-backend/internal/model"
	user "sound-stage-backend/internal/user"

	"gorm.io/gorm"
)

type Room struct {
	model.BaseModel
	Name        string    `gorm:"not null" json:"name" validate:"required"`
	Description string    `json:"description" validate:"required"`
	CreatorID   uint      `json:"creatorID" validate:"required"`
	Creator     user.User `gorm:"foreignKey:CreatorID" json:"creator"`
	DeletedAt   gorm.DeletedAt
}

func (Room) TableName() string {
	return "rooms"
}

type CreateRoomParams struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
	CreatorID   uint   `json:"creatorID" validate:"required"`
}

type UpdateRoomParams struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description" validate:"omitempty"`
}
