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
	FindAnyBy(userID uint, roomID uint) (*RoomUser, error)
	UpdateActivity(ru *RoomUser, activity Activity) error
	HasRoles(userID uint, roomID uint, permissions []role.RoleName) (bool, error)
	ListByRoomID(roomID uint, filter RoomUserFilter, sort listopts.Sort, p listopts.Pagination) ([]RoomUser, error)
	CountByRoomID(roomID uint, filter RoomUserFilter) (int64, error)
	UpdateRole(roomID uint, userID uint, roleID uint) error
	Delete(roomID uint, userID uint) error
	MapByUserAndRoomIDs(userID uint, roomIDs []uint) (map[uint]*RoomUser, error)
	IsBlocked(roomID, userID uint) (bool, error)
	Block(roomID, userID, blockedByID uint) error
	Unblock(roomID, userID uint) error
	ListBlockedByRoomID(roomID uint, p listopts.Pagination) ([]RoomUser, error)
	CountBlockedByRoomID(roomID uint) (int64, error)
}

type Service struct {
	repo        repo
	roleService roleFinder
	revoker     publishRevoker
}

func NewService(r repo, roleService roleFinder, revoker publishRevoker) *Service {
	return &Service{repo: r, roleService: roleService, revoker: revoker}
}

func (s *Service) Rejoin(ru *RoomUser) error {
	return s.repo.UpdateActivity(ru, ActivityJoin)
}

func (s *Service) Create(tx *gorm.DB, userID uint, roomID uint, roleName role.RoleName) (*RoomUser, error) {
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

func (s *Service) FindAnyBy(userID uint, roomID uint) (*RoomUser, error) {
	return s.repo.FindAnyBy(userID, roomID)
}

func (s *Service) IsBlocked(roomID, userID uint) (bool, error) {
	return s.repo.IsBlocked(roomID, userID)
}

func (s *Service) Block(roomID, userID, actorID uint) error {
	if userID == actorID {
		return httpx.ErrForbidden
	}

	actorRoomUser, err := s.repo.FindBy(actorID, roomID)
	if err != nil {
		return err
	}
	if actorRoomUser == nil {
		return httpx.ErrForbidden
	}

	targetRoomUser, err := s.repo.FindAnyBy(userID, roomID)
	if err != nil {
		return err
	}
	if targetRoomUser == nil {
		return httpx.ErrRecordNotFound
	}

	if !role.CanModerate(actorRoomUser.Role.Name, targetRoomUser.Role.Name) {
		return httpx.ErrForbidden
	}

	if err := s.repo.Block(roomID, userID, actorID); err != nil {
		return err
	}

	s.revoker.RevokePublishing(roomID, userID)

	return nil
}

func (s *Service) Unblock(roomID, userID, actorID uint) error {
	actorRoomUser, err := s.repo.FindBy(actorID, roomID)
	if err != nil {
		return err
	}
	if actorRoomUser == nil {
		return httpx.ErrForbidden
	}

	targetRoomUser, err := s.repo.FindAnyBy(userID, roomID)
	if err != nil {
		return err
	}
	if targetRoomUser == nil || !targetRoomUser.IsBlocked {
		return httpx.ErrRecordNotFound
	}

	if !role.CanModerate(actorRoomUser.Role.Name, targetRoomUser.Role.Name) {
		return httpx.ErrForbidden
	}

	return s.repo.Unblock(roomID, userID)
}

func (s *Service) ListBlockedByRoomID(roomID, actorID uint, p listopts.Pagination) ([]RoomUser, int64, error) {
	actorRoomUser, err := s.repo.FindBy(actorID, roomID)
	if err != nil {
		return nil, 0, err
	}
	if actorRoomUser == nil || !actorRoomUser.CanManage() {
		return nil, 0, httpx.ErrForbidden
	}

	users, err := s.repo.ListBlockedByRoomID(roomID, p)
	if err != nil {
		return nil, 0, err
	}
	count, err := s.repo.CountBlockedByRoomID(roomID)
	if err != nil {
		return nil, 0, err
	}
	return users, count, nil
}

func (s *Service) MapByUserAndRoomIDs(userID uint, roomIDs []uint) (map[uint]*RoomUser, error) {
	return s.repo.MapByUserAndRoomIDs(userID, roomIDs)
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

func (s *Service) ListByRoomID(roomID, userID uint, filter RoomUserFilter, sort listopts.Sort, p listopts.Pagination) ([]RoomUser, int64, error) {
	blocked, err := s.repo.IsBlocked(roomID, userID)
	if err != nil {
		return nil, 0, err
	}
	if blocked {
		return nil, 0, httpx.ErrUserBlocked
	}

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

func (s *Service) HasRoles(userID uint, roomID uint, permissions []role.RoleName) (bool, error) {
	return s.repo.HasRoles(userID, roomID, permissions)
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

func (s *Service) DeleteUser(roomID, userID, actorID uint) error {
	if userID == actorID {
		return s.repo.Delete(roomID, userID)
	}

	actorRoomUser, err := s.repo.FindBy(actorID, roomID)
	if err != nil {
		return err
	}
	if actorRoomUser == nil {
		return httpx.ErrForbidden
	}

	targetRoomUser, err := s.repo.FindBy(userID, roomID)
	if err != nil {
		return err
	}
	if targetRoomUser == nil {
		return httpx.ErrRecordNotFound
	}

	if !role.CanModerate(actorRoomUser.Role.Name, targetRoomUser.Role.Name) {
		return httpx.ErrForbidden
	}

	return s.repo.Delete(roomID, userID)
}
