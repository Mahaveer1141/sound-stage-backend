package roomuserblock

import (
	"sound-stage-backend/internal/pkg/listopts"

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

func (r *Repo) ListByRoomID(roomID uint, p listopts.Pagination) ([]RoomUserBlock, error) {
	var blocks []RoomUserBlock
	err := r.db.Where("room_id = ?", roomID).
		Preload("User").
		Preload("BlockedBy").
		Order("id DESC").
		Scopes(p.Scope()).
		Find(&blocks).Error
	return blocks, err
}

func (r *Repo) CountByRoomID(roomID uint) (int64, error) {
	var count int64
	err := r.db.Model(&RoomUserBlock{}).Where("room_id = ?", roomID).Count(&count).Error
	return count, err
}
