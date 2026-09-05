package roomuserblock

import (
	"sound-stage-backend/internal/pkg/httpx"
	"sound-stage-backend/internal/pkg/listopts"
	"sound-stage-backend/internal/role"
	roomuser "sound-stage-backend/internal/room_user"
)

type roomUserService interface {
	FindBy(userID, roomID uint) (*roomuser.RoomUser, error)
	HasRoles(userID, roomID uint, roles []role.RoleName) (bool, error)
}

type repo interface {
	Add(roomID, userID, blockedByID uint) error
	Remove(roomID, userID uint) error
	ListByRoomID(roomID uint, p listopts.Pagination) ([]RoomUserBlock, error)
	CountByRoomID(roomID uint) (int64, error)
}

type Service struct {
	repo        repo
	roomUserSvc roomUserService
}

func NewService(r repo, roomUserSvc roomUserService) *Service {
	return &Service{repo: r, roomUserSvc: roomUserSvc}
}

func (s *Service) Add(roomID, userID, blockedByID uint) error {
	if err := s.canBlock(roomID, userID, blockedByID); err != nil {
		return err
	}
	return s.repo.Add(roomID, userID, blockedByID)
}

func (s *Service) Remove(roomID, userID, blockedByID uint) error {
	if err := s.canBlock(roomID, userID, blockedByID); err != nil {
		return err
	}
	return s.repo.Remove(roomID, userID)
}

func (s *Service) ListByRoomID(roomID, actorID uint, p listopts.Pagination) ([]RoomUserBlock, int64, error) {
	hasPermissions, err := s.roomUserSvc.HasRoles(actorID, roomID, []role.RoleName{role.RoleAdmin, role.RoleOwner})
	if err != nil {
		return nil, 0, err
	}
	if !hasPermissions {
		return nil, 0, httpx.ErrForbidden
	}

	blocks, err := s.repo.ListByRoomID(roomID, p)
	if err != nil {
		return nil, 0, err
	}

	count, err := s.repo.CountByRoomID(roomID)
	if err != nil {
		return nil, 0, err
	}

	return blocks, count, nil
}

func (s *Service) canBlock(roomID, userID, blockedByID uint) error {
	blocker, err := s.roomUserSvc.FindBy(blockedByID, roomID)
	if err != nil {
		return err
	}
	if blocker == nil {
		return httpx.ErrForbidden
	}

	target, err := s.roomUserSvc.FindBy(userID, roomID)
	if err != nil {
		return err
	}
	if target == nil {
		return httpx.ErrRecordNotFound
	}

	if !role.CanModerate(blocker.Role.Name, target.Role.Name) {
		return httpx.ErrForbidden
	}

	return nil
}
