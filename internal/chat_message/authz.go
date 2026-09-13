package chatmessage

import (
	"errors"

	"sound-stage-backend/internal/pkg/httpx"
	"sound-stage-backend/internal/room"
	roomuser "sound-stage-backend/internal/room_user"

	"gorm.io/gorm"
)

type roomUserFinder interface {
	FindBy(userID, roomID uint) (*roomuser.RoomUser, error)
}

type roomFinder interface {
	FindByID(id uint) (*room.Room, error)
}

type Authz struct {
	roomUsers roomUserFinder
	rooms     roomFinder
}

func NewAuthz(roomUsers roomUserFinder, rooms roomFinder) *Authz {
	return &Authz{roomUsers: roomUsers, rooms: rooms}
}

func (a *Authz) ensureChatEnabled(roomID uint) error {
	rm, err := a.rooms.FindByID(roomID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return httpx.ErrForbidden
		}
		return err
	}
	if rm == nil || !rm.IsChatEnabled {
		return httpx.ErrForbidden
	}
	return nil
}

func (a *Authz) CanCreate(input *CreateChatMessageParams) error {
	ru, err := a.roomUsers.FindBy(input.UserID, input.RoomID)
	if err != nil {
		return err
	}
	if ru == nil {
		return httpx.ErrForbidden
	}
	if err := a.ensureChatEnabled(input.RoomID); err != nil {
		return err
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
	return a.ensureChatEnabled(roomID)
}

func (a *Authz) CanUpdatePin(userID, roomID uint) error {
	ru, err := a.roomUsers.FindBy(userID, roomID)
	if err != nil {
		return err
	}
	if ru == nil {
		return httpx.ErrForbidden
	}
	if err := a.ensureChatEnabled(roomID); err != nil {
		return err
	}
	if !ru.IsAdmin() {
		return httpx.ErrForbidden
	}
	return nil
}
