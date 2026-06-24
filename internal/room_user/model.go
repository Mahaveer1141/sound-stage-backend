package roomuser

import (
	model "sound-stage-backend/internal/model"
	"sound-stage-backend/internal/room"
	"sound-stage-backend/internal/user"
	"time"
)

type RoomUser struct {
	model.BaseModel
	UserID       uint
	RoomID       uint
	LastJoinedAt time.Time `json:"lastJoinedAt"`
	LastLeftAt   time.Time `json:"lastLeftAt"`
	IsOnline     bool      `json:"isOnline"`
	Room         room.Room `gorm:"foreignKey:RoomID"`
	User         user.User `gorm:"foreignKey:UserID"`
}

func (RoomUser) TableName() string {
	return "room_users"
}
