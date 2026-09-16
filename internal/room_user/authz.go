package roomuser

import (
	"sound-stage-backend/internal/pkg/httpx"
	"sound-stage-backend/internal/role"
)

type roomUserFinder interface {
	FindBy(userID uint, roomID uint) (*RoomUser, error)
	FindAnyBy(userID uint, roomID uint) (*RoomUser, error)
	HasRoles(userID, roomID uint, permissions []role.RoleName) (bool, error)
}

type Authz struct {
	repo roomUserFinder
}

func NewAuthz(repo roomUserFinder) *Authz {
	return &Authz{repo: repo}
}

func (a *Authz) CanBlock(roomID, actorID, targetID uint) (*RoomUser, error) {
	if targetID == actorID {
		return nil, httpx.ErrForbidden
	}
	_, targetRoomUser, err := a.CanModerate(roomID, actorID, targetID)
	if err != nil {
		return nil, err
	}
	return targetRoomUser, nil
}

func (a *Authz) CanUnblock(roomID, actorID, targetID uint) error {
	_, targetRoomUser, err := a.CanModerate(roomID, actorID, targetID)
	if err != nil {
		return err
	}
	if !targetRoomUser.IsBlocked {
		return httpx.ErrRecordNotFound
	}
	return nil
}

func (a *Authz) CanListBlocked(roomID, actorID uint) error {
	actorRoomUser, err := a.repo.FindBy(actorID, roomID)
	if err != nil {
		return err
	}
	if actorRoomUser == nil || !actorRoomUser.CanManage() {
		return httpx.ErrForbidden
	}
	return nil
}

func (a *Authz) CanDeleteUser(roomID, userID, actorID uint) (*RoomUser, error) {
	if actorID == userID {
		actorRoomUser, err := a.repo.FindBy(actorID, roomID)
		if err != nil {
			return nil, err
		}
		if actorRoomUser == nil {
			return nil, httpx.ErrForbidden
		}
		if actorRoomUser.Role.Name == role.RoleOwner {
			return nil, httpx.ErrForbidden
		}
		return actorRoomUser, nil
	}

	_, targetRoomUser, err := a.CanModerate(roomID, actorID, userID)
	if err != nil {
		return nil, err
	}
	if targetRoomUser.IsBlocked {
		return nil, httpx.ErrUserBlocked
	}
	return targetRoomUser, nil
}

func (a *Authz) CanUpdateRole(roomID, actorID uint, roleName role.RoleName) error {
	ok, err := a.repo.HasRoles(actorID, roomID, role.RoleAssignmentPermissions[roleName])
	if err != nil {
		return err
	}
	if !ok {
		return httpx.ErrForbidden
	}
	return nil
}

func (a *Authz) CanJoinRoom(roomID, userID uint) error {
	ru, err := a.repo.FindBy(userID, roomID)
	if err != nil {
		return err
	}
	if ru == nil {
		return httpx.ErrForbidden
	}
	return nil
}

func (a *Authz) CanSetHandRaised(roomID, userID uint) error {
	ru, err := a.repo.FindAnyBy(userID, roomID)
	if err != nil {
		return err
	}
	if ru == nil {
		return httpx.ErrForbidden
	}
	if ru.IsBlocked {
		return httpx.ErrUserBlocked
	}
	return nil
}

func (a *Authz) CanSetMuted(roomID, actorID, targetID uint, isMuted bool) error {
	if actorID == targetID {
		ru, err := a.repo.FindAnyBy(actorID, roomID)
		if err != nil {
			return err
		}
		if ru == nil {
			return httpx.ErrForbidden
		}
		if ru.IsBlocked {
			return httpx.ErrUserBlocked
		}
		return nil
	}

	if !isMuted {
		return httpx.ErrForbidden
	}

	_, _, err := a.CanModerate(roomID, actorID, targetID)
	return err
}

func (a *Authz) CanListRaisedHands(roomID, userID uint) error {
	return a.CanListUsers(roomID, userID)
}

func (a *Authz) CanListUsers(roomID, userID uint) error {
	ru, err := a.repo.FindAnyBy(userID, roomID)
	if err != nil {
		return err
	}
	if ru == nil {
		return httpx.ErrForbidden
	}
	if ru.IsBlocked {
		return httpx.ErrUserBlocked
	}
	return nil
}

func (a *Authz) CanModerate(roomID, actorID, targetID uint) (*RoomUser, *RoomUser, error) {
	actorRoomUser, err := a.repo.FindBy(actorID, roomID)
	if err != nil {
		return nil, nil, err
	}
	if actorRoomUser == nil {
		return nil, nil, httpx.ErrForbidden
	}

	targetRoomUser, err := a.repo.FindAnyBy(targetID, roomID)
	if err != nil {
		return nil, nil, err
	}
	if targetRoomUser == nil {
		return nil, nil, httpx.ErrRecordNotFound
	}

	if !role.CanModerate(actorRoomUser.Role.Name, targetRoomUser.Role.Name) {
		return nil, nil, httpx.ErrForbidden
	}

	return actorRoomUser, targetRoomUser, nil
}
