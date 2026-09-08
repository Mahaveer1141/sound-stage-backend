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

	ru := RoomUser{RoomID: roomID, UserID: userID, RoleID: roleID, LastJoinedAt: time.Now(), LastLeftAt: time.Now()}
	if err := tx.Create(&ru).Error; err != nil {
		return nil, err
	}
	return &ru, nil
}

func (r *Repo) FindBy(userID uint, roomID uint) (*RoomUser, error) {
	var ru RoomUser
	result := r.db.Where("user_id = ? AND room_id = ? AND is_blocked = ?", userID, roomID, false).
		Preload("User").Preload("Role").First(&ru)
	return gormutil.NilIfNotFound(&ru, result.Error)
}

func (r *Repo) FindAnyBy(userID uint, roomID uint) (*RoomUser, error) {
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
		Where("room_users.user_id = ? AND room_users.room_id = ? AND room_users.is_blocked = ? AND roles.name IN (?)",
			userID, roomID, false, roles).
		Count(&count).Error

	return count > 0, err
}

func (r *Repo) ListByRoomID(roomID uint, filter RoomUserFilter, sort listopts.Sort, p listopts.Pagination) ([]RoomUser, error) {
	var roomUsers []RoomUser
	err := r.db.
		Preload("User").
		Preload("Role").
		Where("room_users.room_id = ? AND room_users.is_blocked = ?", roomID, false).
		Scopes(Filters(filter), Sort(sort), p.Scope()).
		Find(&roomUsers).Error
	return roomUsers, err
}

func (r *Repo) ListByUserIDs(roomID uint, userIDs []uint) ([]RoomUser, error) {
	var roomUsers []RoomUser
	if len(userIDs) == 0 {
		return roomUsers, nil
	}
	err := r.db.
		Preload("User").
		Preload("Role").
		Where("room_id = ? AND user_id IN ? AND is_blocked = ?", roomID, userIDs, false).
		Scopes(SortByUserIDs(userIDs)).
		Find(&roomUsers).Error
	return roomUsers, err
}

func (r *Repo) CountByRoomID(roomID uint, filter RoomUserFilter) (int64, error) {
	var count int64
	err := r.db.
		Table("room_users").
		Where("room_users.room_id = ? AND room_users.is_blocked = ?", roomID, false).
		Scopes(Filters(filter)).
		Count(&count).Error
	return count, err
}

func (r *Repo) CountByRoomIDs(roomIDs []uint, filter RoomUserFilter) (map[uint]int64, error) {
	if len(roomIDs) == 0 {
		return map[uint]int64{}, nil
	}
	var rows []struct {
		RoomID uint
		Count  int64
	}
	err := r.db.
		Table("room_users").
		Select("room_users.room_id as room_id, count(*) as count").
		Where("room_users.room_id IN ? AND room_users.is_blocked = ?", roomIDs, false).
		Scopes(Filters(filter)).
		Group("room_users.room_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[uint]int64, len(roomIDs))
	for _, row := range rows {
		out[row.RoomID] = row.Count
	}
	return out, nil
}

func (r *Repo) UpdateRole(roomID uint, userID uint, roleID uint) error {
	return r.db.
		Table("room_users").
		Where("room_id = ? AND user_id = ?", roomID, userID).
		Update("role_id", roleID).Error
}

func (r *Repo) Delete(roomID uint, userID uint) error {
	return r.db.
		Where("room_id = ? AND user_id = ?", roomID, userID).
		Delete(&RoomUser{}).Error
}

func (r *Repo) MapByUserAndRoomIDs(userID uint, roomIDs []uint) (map[uint]*RoomUser, error) {
	var roomUsers []RoomUser
	err := r.db.
		Preload("User").
		Preload("Role").
		Where("user_id = ? AND room_id IN ? AND is_blocked = ?", userID, roomIDs, false).
		Find(&roomUsers).Error
	if err != nil {
		return nil, err
	}
	out := make(map[uint]*RoomUser, len(roomUsers))
	for i := range roomUsers {
		out[roomUsers[i].RoomID] = &roomUsers[i]
	}
	return out, nil
}

func (r *Repo) IsBlocked(roomID, userID uint) (bool, error) {
	var count int64
	err := r.db.Model(&RoomUser{}).
		Where("room_id = ? AND user_id = ? AND is_blocked = ?", roomID, userID, true).
		Count(&count).Error
	return count > 0, err
}

func (r *Repo) Block(roomID, userID, blockedByID uint) error {
	return r.db.Model(&RoomUser{}).
		Where("room_id = ? AND user_id = ?", roomID, userID).
		Updates(map[string]any{
			"is_blocked":    true,
			"blocked_by_id": blockedByID,
			"last_left_at":  time.Now(),
			"is_online":     false,
		}).Error
}

func (r *Repo) Unblock(roomID, userID uint) error {
	return r.db.Model(&RoomUser{}).
		Where("room_id = ? AND user_id = ?", roomID, userID).
		Updates(map[string]any{
			"is_blocked":    false,
			"blocked_by_id": nil,
		}).Error
}

func (r *Repo) ListBlockedByRoomID(roomID uint, p listopts.Pagination) ([]RoomUser, error) {
	var roomUsers []RoomUser
	err := r.db.
		Preload("User").
		Where("room_id = ? AND is_blocked = ?", roomID, true).
		Order("id DESC").
		Scopes(p.Scope()).
		Find(&roomUsers).Error
	return roomUsers, err
}

func (r *Repo) CountBlockedByRoomID(roomID uint) (int64, error) {
	var count int64
	err := r.db.Model(&RoomUser{}).
		Where("room_id = ? AND is_blocked = ?", roomID, true).
		Count(&count).Error
	return count, err
}
