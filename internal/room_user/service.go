package roomuser

import (
	"sound-stage-backend/internal/pkg/httpx"
	"sound-stage-backend/internal/pkg/listopts"
	"sound-stage-backend/internal/role"

	"gorm.io/gorm"
)

type roleFinder interface {
	FindByName(name role.RoleName) (*role.Role, error)
}

type publishRevoker interface {
	RevokePublishing(roomID uint, userID uint)
}

type repo interface {
	Create(tx *gorm.DB, userID uint, roomID uint, roleID uint) (*RoomUser, error)
	FindBy(userID uint, roomID uint) (*RoomUser, error)
	UpdateActivity(ru *RoomUser, activity Activity) error
	HasRoles(userID uint, roomID uint, permissions []role.RoleName) (bool, error)
	ListByRoomID(roomID uint, filter RoomUserFilter, sort listopts.Sort, p listopts.Pagination) ([]RoomUser, error)
	CountByRoomID(roomID uint, filter RoomUserFilter) (int64, error)
	UpdateRole(roomID uint, userID uint, roleID uint) error
}

type Service struct {
	repo        repo
	roleService roleFinder
	revoker     publishRevoker
}

func NewService(r repo, roleService roleFinder, revoker publishRevoker) *Service {
	return &Service{repo: r, roleService: roleService, revoker: revoker}
}

func (s *Service) AddUser(userID uint, roomID uint, roleName role.RoleName) (*RoomUser, error) {
	return s.AddUserWithTx(nil, userID, roomID, roleName)
}

func (s *Service) AddUserWithTx(tx *gorm.DB, userID uint, roomID uint, roleName role.RoleName) (*RoomUser, error) {
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

func (s *Service) FindBy(userID uint, roomID uint) (*RoomUser, error) {
	return s.repo.FindBy(userID, roomID)
}

func (s *Service) RemoveUser(userID uint, roomID uint) error {
	ru, err := s.repo.FindBy(userID, roomID)
	if err != nil {
		return err
	}
	if ru == nil {
		return httpx.ErrRecordNotFound
	}
	return s.repo.UpdateActivity(ru, ActivityLeave)
}

func (s *Service) ListByRoomID(roomID uint, filter RoomUserFilter, sort listopts.Sort, p listopts.Pagination) ([]RoomUser, int64, error) {
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

func (s *Service) UpdateRole(roomID uint, userID uint, roleName role.RoleName, actorID uint) error {
	hasPermission, err := s.repo.HasRoles(actorID, roomID, role.RoleAssignmentPermissions[roleName])
	if err != nil {
		return err
	}
	if !hasPermission {
		return httpx.ErrForbidden
	}
	r, err := s.roleService.FindByName(roleName)
	if err != nil {
		return err
	}
	if err := s.repo.UpdateRole(roomID, userID, r.ID); err != nil {
		return err
	}

	if roleName == role.RoleListener {
		s.revoker.RevokePublishing(roomID, userID)
	}

	return nil
}
