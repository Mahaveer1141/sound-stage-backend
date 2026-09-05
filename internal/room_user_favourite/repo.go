package roomuserfavourite

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

func (r *Repo) Add(userID, roomID uint) error {
	favourite := RoomUserFavourite{UserID: userID, RoomID: roomID}
	return r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&favourite).Error
}

func (r *Repo) Remove(userID, roomID uint) error {
	return r.db.Where("user_id = ? AND room_id = ?", userID, roomID).Delete(&RoomUserFavourite{}).Error
}

func (r *Repo) ListUserFavourites(userID uint, p listopts.Pagination) ([]RoomUserFavourite, error) {
	var favourites []RoomUserFavourite
	err := r.db.Where("user_id = ?", userID).
		Preload("Room").
		Order("id DESC").
		Scopes(p.Scope()).
		Find(&favourites).Error
	return favourites, err
}

func (r *Repo) CountByUserID(userID uint) (int64, error) {
	var count int64
	err := r.db.Model(&RoomUserFavourite{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}
