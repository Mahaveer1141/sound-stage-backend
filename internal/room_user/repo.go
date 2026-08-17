package roomuser

import (
	"sound-stage-backend/internal/pkg/gormutil"
	"sound-stage-backend/internal/pkg/listopts"
	"sound-stage-backend/internal/role"
	"time"

	"gorm.io/gorm"
)

type Activity string

const (
	ActivityJoin  Activity = "join"
	ActivityLeave Activity = "leave"
)

type Repo struct {
	db *gorm.DB
}

func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) Create(tx *gorm.DB, userID uint, roomID uint, roleID uint) (*RoomUser, error) {
	if tx == nil {
		tx = r.db
	}

	ru := RoomUser{RoomID: roomID, UserID: userID, RoleID: roleID}
	if err := tx.Create(&ru).Error; err != nil {
		return nil, err
	}
	return &ru, nil
}

func (r *Repo) FindBy(userID uint, roomID uint) (*RoomUser, error) {
	var ru RoomUser
	result := r.db.Where("user_id = ? AND room_id = ?", userID, roomID).
		Preload("User").Preload("Role").First(&ru)
	return gormutil.NilIfNotFound(&ru, result.Error)
}

func (r *Repo) UpdateActivity(ru *RoomUser, activity Activity) error {
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
				"is_online":    false,
			}).Error
	}
	return err
}

func (r *Repo) HasRoles(userID, roomID uint, roles []role.RoleName) (bool, error) {
	var count int64
	err := r.db.
		Table("room_users").
		Joins("JOIN roles ON roles.id = room_users.role_id").
		Where("room_users.user_id = ? AND room_users.room_id = ? AND roles.name IN (?)",
			userID, roomID, roles).
		Count(&count).Error

	return count > 0, err
}

func (r *Repo) ListByRoomID(roomID uint, filter RoomUserFilter, sort listopts.Sort, p listopts.Pagination) ([]RoomUser, error) {
	var roomUsers []RoomUser
	err := r.db.
		Preload("User").
		Preload("Role").
		Where("room_users.room_id = ?", roomID).
		Scopes(Filters(filter), Sort(sort), p.Scope()).
		Find(&roomUsers).Error
	return roomUsers, err
}

func (r *Repo) CountByRoomID(roomID uint, filter RoomUserFilter) (int64, error) {
	var count int64
	err := r.db.
		Table("room_users").
		Where("room_users.room_id = ?", roomID).
		Scopes(Filters(filter)).
		Count(&count).Error
	return count, err
}

func (r *Repo) UpdateRole(roomID uint, userID uint, roleID uint) error {
	return r.db.
		Table("room_users").
		Where("room_id = ? AND user_id = ?", roomID, userID).
		Update("role_id", roleID).Error
}
