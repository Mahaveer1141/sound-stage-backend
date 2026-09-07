package room

import (
	"sound-stage-backend/internal/pkg/httpx"
	"sound-stage-backend/internal/role"
	roomuser "sound-stage-backend/internal/room_user"
)

type roomUserAuthzDeps interface {
	FindBy(userID, roomID uint) (*roomuser.RoomUser, error)
	HasRoles(userID, roomID uint, permissions []role.RoleName) (bool, error)
	IsBlocked(roomID, userID uint) (bool, error)
}

type Authz struct {
	roomUsers roomUserAuthzDeps
}

func NewAuthz(roomUsers roomUserAuthzDeps) *Authz {
	return &Authz{roomUsers: roomUsers}
}

func (a *Authz) CanView(roomID, userID uint) error {
	ru, err := a.roomUsers.FindBy(userID, roomID)
	if err != nil {
		return err
	}
	if ru == nil {
		return httpx.ErrForbidden
	}
	return nil
}

func (a *Authz) CanAddRoomUser(roomID, userID uint) error {
	blocked, err := a.roomUsers.IsBlocked(roomID, userID)
	if err != nil {
		return err
	}
	if blocked {
		return httpx.ErrUserBlocked
	}
	return nil
}

func (a *Authz) CanUpdate(roomID, userID uint) error {
	ok, err := a.roomUsers.HasRoles(userID, roomID, []role.RoleName{role.RoleOwner, role.RoleAdmin})
	if err != nil {
		return err
	}
	if !ok {
		return httpx.ErrForbidden
	}
	return nil
}
