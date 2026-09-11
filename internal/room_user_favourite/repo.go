package roomuserfavourite

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

func (r *Repo) Add(userID, roomID uint) error {
	favourite := RoomUserFavourite{UserID: userID, RoomID: roomID}
	return r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&favourite).Error
}

func (r *Repo) Remove(userID, roomID uint) error {
	return r.db.Where("user_id = ? AND room_id = ?", userID, roomID).Delete(&RoomUserFavourite{}).Error
}

func (r *Repo) FindRoomIDsByUserID(userID uint, roomIDs []uint) ([]uint, error) {
	var ids []uint
	err := r.db.Model(&RoomUserFavourite{}).
		Where("user_id = ? AND room_id IN ?", userID, roomIDs).
		Pluck("room_id", &ids).Error
	return ids, err
}
