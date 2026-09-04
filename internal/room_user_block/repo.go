package roomuserblock

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repo struct {
	db *gorm.DB
}

func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) Add(roomID, userID, blockedByID uint) error {
	block := RoomUserBlock{RoomID: roomID, UserID: userID, BlockedByID: blockedByID}
	return r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&block).Error
}

func (r *Repo) Remove(roomID, userID uint) error {
	return r.db.Where("room_id = ? AND user_id = ?", roomID, userID).Delete(&RoomUserBlock{}).Error
}
