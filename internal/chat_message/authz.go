package chatmessage

import (
	"sound-stage-backend/internal/pkg/httpx"
	roomuser "sound-stage-backend/internal/room_user"
)

type roomUserFinder interface {
	FindBy(userID, roomID uint) (*roomuser.RoomUser, error)
}

type Authz struct {
	roomUsers roomUserFinder
}

func NewAuthz(roomUsers roomUserFinder) *Authz {
	return &Authz{roomUsers: roomUsers}
}

func (a *Authz) CanCreate(input *CreateChatMessageParams) error {
	ru, err := a.roomUsers.FindBy(input.UserID, input.RoomID)
	if err != nil {
		return err
	}
	if ru == nil {
		return httpx.ErrForbidden
	}
	if input.IsPinned && !ru.IsAdmin() {
		return httpx.ErrForbidden
	}
	return nil
}

func (a *Authz) CanList(userID, roomID uint) error {
	ru, err := a.roomUsers.FindBy(userID, roomID)
	if err != nil {
		return err
	}
	if ru == nil {
		return httpx.ErrForbidden
	}
	return nil
}
