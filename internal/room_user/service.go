package roomuser

import (
	"sound-stage-backend/internal/pkg/httpx"
	"sound-stage-backend/internal/pkg/listopts"
	"sound-stage-backend/internal/role"

	"gorm.io/gorm"
)

type Service interface {
	AddUser(userID uint, roomID uint, roleName string) (*RoomUser, error)
	AddUserWithTx(tx *gorm.DB, userID uint, roomID uint, roleName string) (*RoomUser, error)
	RemoveUser(userID uint, roomID uint) error
	HasPermission(userID uint, roomID uint, permission string) (bool, error)
	SetRole(userID uint, roomID uint, roleName string) error
	ListByRoomID(roomID uint, filter RoomUserFilter, sort listopts.Sort, p listopts.Pagination) ([]RoomUser, int64, error)
	FindBy(userID uint, roomID uint) (*RoomUser, error)
	UpdateRole(roomID uint, userID uint, role string) error
}

type service struct {
	repo        Repo
	roleService role.Service
}

func NewService(repo Repo, roleService role.Service) Service {
	return &service{repo: repo, roleService: roleService}
}

func (s *service) AddUser(userID uint, roomID uint, roleName string) (*RoomUser, error) {
	return s.AddUserWithTx(nil, userID, roomID, roleName)
}

func (s *service) AddUserWithTx(tx *gorm.DB, userID uint, roomID uint, roleName string) (*RoomUser, error) {
	ru, err := s.repo.FindBy(userID, roomID)
	if err != nil {
		return nil, err
	}
	if ru != nil {
		if err := s.repo.UpdateActivity(ru, ActivityJoin); err != nil {
			return nil, err
		}
		return ru, nil
	}
	if roleName == "" {
		roleName = role.RoleListener
	}
	role, err := s.roleService.FindByName(roleName)
	if err != nil {
		return nil, err
	}
	return s.repo.Create(tx, userID, roomID, role.ID)
}

func (s *service) FindBy(userID uint, roomID uint) (*RoomUser, error) {
	return s.repo.FindBy(userID, roomID)
}

func (s *service) RemoveUser(userID uint, roomID uint) error {
	ru, err := s.repo.FindBy(userID, roomID)
	if err != nil {
		return err
	}
	if ru == nil {
		return httpx.ErrRecordNotFound
	}
	return s.repo.UpdateActivity(ru, ActivityLeave)
}

func (s *service) HasPermission(userID, roomID uint, permission string) (bool, error) {
	return s.repo.HasPermission(userID, roomID, permission)
}

func (s *service) SetRole(userID, roomID uint, roleName string) error {
	return s.repo.SetRole(userID, roomID, roleName)
}

func (s *service) ListByRoomID(roomID uint, filter RoomUserFilter, sort listopts.Sort, p listopts.Pagination) ([]RoomUser, int64, error) {
	users, err := s.repo.ListByRoomID(roomID, filter, sort, p)
	if err != nil {
		return nil, 0, err
	}
	count, err := s.repo.CountByRoomID(roomID, filter)
	if err != nil {
		return nil, 0, err
	}
	return users, count, nil
}

func (s *service) UpdateRole(roomID uint, userID uint, role string) error {
	r, err := s.roleService.FindByName(role)
	if err != nil {
		return err
	}
	return s.repo.UpdateRole(roomID, userID, r.ID)
}
