package roomuser

import (
	model "sound-stage-backend/internal/model"
	"time"
)

type RoomUser struct {
	model.BaseModel
	UserID       uint
	RoomID       uint
	LastJoinedAt time.Time `json:"lastJoinedAt"`
	LastLeftAt   time.Time `json:"lastLeftAt"`
	IsOnline     bool      `json:"isOnline"`
}

func (RoomUser) TableName() string {
	return "room_users"
}
