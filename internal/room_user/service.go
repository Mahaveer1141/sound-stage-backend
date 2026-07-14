package roomuser

import (
	"sound-stage-backend/internal/pkg/httpx"
	"sound-stage-backend/internal/role"

	"gorm.io/gorm"
)

type Service interface {
	AddUser(userID uint, roomID uint, roleName string) (*RoomUser, error)
	AddUserWithTx(tx *gorm.DB, userID uint, roomID uint, roleName string) (*RoomUser, error)
	RemoveUser(userID uint, roomID uint) error
	HasPermission(userID uint, roomID uint, permission string) (bool, error)
	SetRole(userID uint, roomID uint, roleName string) error
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
