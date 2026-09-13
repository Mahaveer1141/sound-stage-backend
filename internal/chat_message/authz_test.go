package chatmessage

import (
	"errors"
	"sound-stage-backend/internal/pkg/httpx"
	"sound-stage-backend/internal/role"
	"sound-stage-backend/internal/room"
	roomuser "sound-stage-backend/internal/room_user"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAuthz_CanList(t *testing.T) {
	t.Run("success: member can list when chat is enabled", func(t *testing.T) {
		ru := new(mockRoomUserService)
		rooms := new(mockRoomFinder)
		ru.On("FindBy", uint(2), uint(1)).Return(&roomuser.RoomUser{Role: role.Role{Name: role.RoleListener}}, nil)
		rooms.On("FindByID", uint(1)).Return(&room.Room{IsChatEnabled: true}, nil)
		authz := NewAuthz(ru, rooms)

		err := authz.CanList(2, 1)

		require.NoError(t, err)
		ru.AssertExpectations(t)
		rooms.AssertExpectations(t)
	})

	t.Run("failure: chat disabled is forbidden", func(t *testing.T) {
		ru := new(mockRoomUserService)
		rooms := new(mockRoomFinder)
		ru.On("FindBy", uint(2), uint(1)).Return(&roomuser.RoomUser{Role: role.Role{Name: role.RoleListener}}, nil)
		rooms.On("FindByID", uint(1)).Return(&room.Room{IsChatEnabled: false}, nil)
		authz := NewAuthz(ru, rooms)

		err := authz.CanList(2, 1)

		require.ErrorIs(t, err, httpx.ErrForbidden)
		ru.AssertExpectations(t)
		rooms.AssertExpectations(t)
	})

	t.Run("failure: non-member is forbidden", func(t *testing.T) {
		ru := new(mockRoomUserService)
		rooms := new(mockRoomFinder)
		ru.On("FindBy", uint(2), uint(1)).Return(nil, nil)
		authz := NewAuthz(ru, rooms)

		err := authz.CanList(2, 1)

		require.ErrorIs(t, err, httpx.ErrForbidden)
		ru.AssertExpectations(t)
		rooms.AssertNotCalled(t, "FindByID", uint(1))
	})

	t.Run("failure: FindBy error is propagated", func(t *testing.T) {
		ru := new(mockRoomUserService)
		rooms := new(mockRoomFinder)
		findErr := errors.New("db down")
		ru.On("FindBy", uint(2), uint(1)).Return(nil, findErr)
		authz := NewAuthz(ru, rooms)

		err := authz.CanList(2, 1)

		require.ErrorIs(t, err, findErr)
		ru.AssertExpectations(t)
	})

	t.Run("failure: room lookup error is propagated", func(t *testing.T) {
		ru := new(mockRoomUserService)
		rooms := new(mockRoomFinder)
		roomErr := errors.New("db down")
		ru.On("FindBy", uint(2), uint(1)).Return(&roomuser.RoomUser{Role: role.Role{Name: role.RoleListener}}, nil)
		rooms.On("FindByID", uint(1)).Return(nil, roomErr)
		authz := NewAuthz(ru, rooms)

		err := authz.CanList(2, 1)

		require.ErrorIs(t, err, roomErr)
		rooms.AssertExpectations(t)
	})
}

func TestAuthz_CanCreate(t *testing.T) {
	t.Run("success: member can create when chat is enabled", func(t *testing.T) {
		ru := new(mockRoomUserService)
		rooms := new(mockRoomFinder)
		ru.On("FindBy", uint(2), uint(1)).Return(&roomuser.RoomUser{Role: role.Role{Name: role.RoleListener}}, nil)
		rooms.On("FindByID", uint(1)).Return(&room.Room{IsChatEnabled: true}, nil)
		authz := NewAuthz(ru, rooms)

		err := authz.CanCreate(&CreateChatMessageParams{RoomID: 1, UserID: 2, Content: "hi"})

		require.NoError(t, err)
		ru.AssertExpectations(t)
		rooms.AssertExpectations(t)
	})

	t.Run("success: admin can pin", func(t *testing.T) {
		ru := new(mockRoomUserService)
		rooms := new(mockRoomFinder)
		ru.On("FindBy", uint(2), uint(1)).Return(&roomuser.RoomUser{Role: role.Role{Name: role.RoleAdmin}}, nil)
		rooms.On("FindByID", uint(1)).Return(&room.Room{IsChatEnabled: true}, nil)
		authz := NewAuthz(ru, rooms)

		err := authz.CanCreate(&CreateChatMessageParams{RoomID: 1, UserID: 2, Content: "hi", IsPinned: true})

		require.NoError(t, err)
		ru.AssertExpectations(t)
		rooms.AssertExpectations(t)
	})

	t.Run("failure: non-admin cannot pin", func(t *testing.T) {
		ru := new(mockRoomUserService)
		rooms := new(mockRoomFinder)
		ru.On("FindBy", uint(2), uint(1)).Return(&roomuser.RoomUser{Role: role.Role{Name: role.RoleListener}}, nil)
		rooms.On("FindByID", uint(1)).Return(&room.Room{IsChatEnabled: true}, nil)
		authz := NewAuthz(ru, rooms)

		err := authz.CanCreate(&CreateChatMessageParams{RoomID: 1, UserID: 2, Content: "hi", IsPinned: true})

		require.ErrorIs(t, err, httpx.ErrForbidden)
		ru.AssertExpectations(t)
		rooms.AssertExpectations(t)
	})

	t.Run("failure: chat disabled is forbidden", func(t *testing.T) {
		ru := new(mockRoomUserService)
		rooms := new(mockRoomFinder)
		ru.On("FindBy", uint(2), uint(1)).Return(&roomuser.RoomUser{Role: role.Role{Name: role.RoleListener}}, nil)
		rooms.On("FindByID", uint(1)).Return(&room.Room{IsChatEnabled: false}, nil)
		authz := NewAuthz(ru, rooms)

		err := authz.CanCreate(&CreateChatMessageParams{RoomID: 1, UserID: 2, Content: "hi"})

		require.ErrorIs(t, err, httpx.ErrForbidden)
		ru.AssertExpectations(t)
		rooms.AssertExpectations(t)
	})

	t.Run("failure: non-member is forbidden", func(t *testing.T) {
		ru := new(mockRoomUserService)
		rooms := new(mockRoomFinder)
		ru.On("FindBy", uint(2), uint(1)).Return(nil, nil)
		authz := NewAuthz(ru, rooms)

		err := authz.CanCreate(&CreateChatMessageParams{RoomID: 1, UserID: 2, Content: "hi"})

		require.ErrorIs(t, err, httpx.ErrForbidden)
		ru.AssertExpectations(t)
		rooms.AssertNotCalled(t, "FindByID", uint(1))
	})
}

func TestAuthz_CanUpdatePin(t *testing.T) {
	t.Run("success: admin can update pin when chat is enabled", func(t *testing.T) {
		ru := new(mockRoomUserService)
		rooms := new(mockRoomFinder)
		ru.On("FindBy", uint(2), uint(1)).Return(&roomuser.RoomUser{Role: role.Role{Name: role.RoleAdmin}}, nil)
		rooms.On("FindByID", uint(1)).Return(&room.Room{IsChatEnabled: true}, nil)
		authz := NewAuthz(ru, rooms)

		err := authz.CanUpdatePin(2, 1)

		require.NoError(t, err)
		ru.AssertExpectations(t)
		rooms.AssertExpectations(t)
	})

	t.Run("failure: non-admin is forbidden", func(t *testing.T) {
		ru := new(mockRoomUserService)
		rooms := new(mockRoomFinder)
		ru.On("FindBy", uint(2), uint(1)).Return(&roomuser.RoomUser{Role: role.Role{Name: role.RoleListener}}, nil)
		rooms.On("FindByID", uint(1)).Return(&room.Room{IsChatEnabled: true}, nil)
		authz := NewAuthz(ru, rooms)

		err := authz.CanUpdatePin(2, 1)

		require.ErrorIs(t, err, httpx.ErrForbidden)
		ru.AssertExpectations(t)
		rooms.AssertExpectations(t)
	})

	t.Run("failure: chat disabled is forbidden", func(t *testing.T) {
		ru := new(mockRoomUserService)
		rooms := new(mockRoomFinder)
		ru.On("FindBy", uint(2), uint(1)).Return(&roomuser.RoomUser{Role: role.Role{Name: role.RoleAdmin}}, nil)
		rooms.On("FindByID", uint(1)).Return(&room.Room{IsChatEnabled: false}, nil)
		authz := NewAuthz(ru, rooms)

		err := authz.CanUpdatePin(2, 1)

		require.ErrorIs(t, err, httpx.ErrForbidden)
		ru.AssertExpectations(t)
		rooms.AssertExpectations(t)
	})

	t.Run("failure: non-member is forbidden", func(t *testing.T) {
		ru := new(mockRoomUserService)
		rooms := new(mockRoomFinder)
		ru.On("FindBy", uint(2), uint(1)).Return(nil, nil)
		authz := NewAuthz(ru, rooms)

		err := authz.CanUpdatePin(2, 1)

		require.ErrorIs(t, err, httpx.ErrForbidden)
		ru.AssertExpectations(t)
		rooms.AssertNotCalled(t, "FindByID", uint(1))
	})
}
