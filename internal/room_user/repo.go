package roomuser

import (
	"sound-stage-backend/internal/pkg/listopts"
	"time"

	"gorm.io/gorm"
)

type Activity string

const (
	ActivityJoin  Activity = "join"
	ActivityLeave Activity = "leave"
)

type Repo interface {
	Create(tx *gorm.DB, userID uint, roomID uint, roleID uint) (*RoomUser, error)
	FindBy(userID uint, roomID uint) (*RoomUser, error)
	UpdateActivity(ru *RoomUser, activity Activity) error
	HasPermission(userID uint, roomID uint, permission string) (bool, error)
	SetRole(userID uint, roomID uint, roleName string) error
	ListByRoomID(roomID uint, filter RoomUserFilter, sort listopts.Sort, p listopts.Pagination) ([]RoomUser, error)
	CountByRoomID(roomID uint, filter RoomUserFilter) (int64, error)
	UpdateRole(roomID uint, userID uint, roleID uint) error
}

type repo struct {
	db *gorm.DB
}

func NewRepo(db *gorm.DB) Repo {
	return &repo{db: db}
}

func (r *repo) Create(tx *gorm.DB, userID uint, roomID uint, roleID uint) (*RoomUser, error) {
	if tx == nil {
		tx = r.db
	}

	ru := RoomUser{RoomID: roomID, UserID: userID, RoleID: roleID}
	if err := tx.Create(&ru).Error; err != nil {
		return nil, err
	}
	return &ru, nil
}

func (r *repo) FindBy(userID uint, roomID uint) (*RoomUser, error) {
	var ru RoomUser
	result := r.db.Where("user_id = ? AND room_id = ?", userID, roomID).Preload("Role").First(&ru)
	return &ru, result.Error
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
				"is_online":    false,
			}).Error
	}
	return err
}

func (r *repo) HasPermission(userID, roomID uint, permission string) (bool, error) {
	var count int64
	err := r.db.
		Table("room_users").
		Joins("JOIN role_permissions ON role_permissions.role_id = room_users.role_id").
		Joins("JOIN permissions ON permissions.id = role_permissions.permission_id").
		Where("room_users.user_id = ? AND room_users.room_id = ? AND permissions.name = ?", userID, roomID, permission).
		Count(&count).Error

	return count > 0, err
}

func (r *repo) SetRole(userID, roomID uint, roleName string) error {
	return r.SetRoleTx(r.db, userID, roomID, roleName)
}

func (r *repo) SetRoleTx(tx *gorm.DB, userID, roomID uint, roleName string) error {
	return tx.
		Table("room_users").
		Where("user_id = ? AND room_id = ?", userID, roomID).
		Update("role_id", tx.Table("roles").Select("id").Where("name = ?", roleName)).Error
}

func (r *repo) ListByRoomID(roomID uint, filter RoomUserFilter, sort listopts.Sort, p listopts.Pagination) ([]RoomUser, error) {
	var roomUsers []RoomUser
	err := r.db.
		Preload("User").
		Preload("Role").
		Where("room_users.room_id = ?", roomID).
		Scopes(Filters(filter), Sort(sort), p.Scope()).
		Find(&roomUsers).Error
	return roomUsers, err
}

func (r *repo) CountByRoomID(roomID uint, filter RoomUserFilter) (int64, error) {
	var count int64
	err := r.db.
		Table("room_users").
		Where("room_users.room_id = ?", roomID).
		Scopes(Filters(filter)).
		Count(&count).Error
	return count, err
}

func (r *repo) UpdateRole(roomID uint, userID uint, roleID uint) error {
	return r.db.
		Table("room_users").
		Where("room_id = ? AND user_id = ?", roomID, userID).
		Update("role_id", roleID).Error
}
