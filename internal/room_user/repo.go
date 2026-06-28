package roomuser

import (
	"sound-stage-backend/internal/pkg/gormutil"
	"time"

	"gorm.io/gorm"
)

type Activity string

const (
	ActivityJoin  Activity = "join"
	ActivityLeave Activity = "leave"
)

type Repo interface {
	Create(userID uint, roomID uint) (*RoomUser, error)
	FindBy(userID uint, roomID uint) (*RoomUser, error)
	UpdateActivity(ru *RoomUser, activity Activity) error
}

type repo struct {
	db *gorm.DB
}

func NewRepo(db *gorm.DB) Repo {
	return &repo{db: db}
}

func (r *repo) Create(userID uint, roomID uint) (*RoomUser, error) {
	user := RoomUser{
		RoomID: roomID,
		UserID: userID,
	}
	if err := r.db.Create(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *repo) FindBy(userID uint, roomID uint) (*RoomUser, error) {
	var ru *RoomUser
	result := r.db.Where("user_id = ? AND room_id = ?", userID, roomID).First(&ru)
	ru, err := gormutil.NilIfNotFound(ru, result.Error)
	if err != nil {
		return nil, err
	}
	return ru, nil
}

func (r *repo) UpdateActivity(ru *RoomUser, activity Activity) error {
	var err error
	if activity == ActivityJoin {
		err = r.db.Model(ru).
			Updates(map[string]any{
				"last_joined_at": time.Now(),
				"is_online":      true,
			}).Error
	} else {
		err = r.db.Model(ru).
			Updates(map[string]any{
				"last_left_at": time.Now(),
				"is_online":    true,
			}).Error
	}
	return err
}
