package chatmessage

import (
	"errors"
	"sound-stage-backend/internal/pkg/httpx"
	"sound-stage-backend/internal/role"
	roomuser "sound-stage-backend/internal/room_user"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAuthz_CanList(t *testing.T) {
	t.Run("success: member can list", func(t *testing.T) {
		ru := new(mockRoomUserService)
		ru.On("FindBy", uint(2), uint(1)).Return(&roomuser.RoomUser{Role: role.Role{Name: role.RoleListener}}, nil)
		authz := NewAuthz(ru)

		err := authz.CanList(2, 1)

		require.NoError(t, err)
		ru.AssertExpectations(t)
	})

	t.Run("failure: non-member is forbidden", func(t *testing.T) {
		ru := new(mockRoomUserService)
		ru.On("FindBy", uint(2), uint(1)).Return(nil, nil)
		authz := NewAuthz(ru)

		err := authz.CanList(2, 1)

		require.ErrorIs(t, err, httpx.ErrForbidden)
		ru.AssertExpectations(t)
	})

	t.Run("failure: FindBy error is propagated", func(t *testing.T) {
		ru := new(mockRoomUserService)
		findErr := errors.New("db down")
		ru.On("FindBy", uint(2), uint(1)).Return(nil, findErr)
		authz := NewAuthz(ru)

		err := authz.CanList(2, 1)

		require.ErrorIs(t, err, findErr)
		ru.AssertExpectations(t)
	})
}

func TestAuthz_CanCreate(t *testing.T) {
	t.Run("success: member can create", func(t *testing.T) {
		ru := new(mockRoomUserService)
		ru.On("FindBy", uint(2), uint(1)).Return(&roomuser.RoomUser{Role: role.Role{Name: role.RoleListener}}, nil)
		authz := NewAuthz(ru)

		err := authz.CanCreate(&CreateChatMessageParams{RoomID: 1, UserID: 2, Content: "hi"})

		require.NoError(t, err)
		ru.AssertExpectations(t)
	})

	t.Run("success: admin can pin", func(t *testing.T) {
		ru := new(mockRoomUserService)
		ru.On("FindBy", uint(2), uint(1)).Return(&roomuser.RoomUser{Role: role.Role{Name: role.RoleAdmin}}, nil)
		authz := NewAuthz(ru)

		err := authz.CanCreate(&CreateChatMessageParams{RoomID: 1, UserID: 2, Content: "hi", IsPinned: true})

		require.NoError(t, err)
		ru.AssertExpectations(t)
	})

	t.Run("failure: non-admin cannot pin", func(t *testing.T) {
		ru := new(mockRoomUserService)
		ru.On("FindBy", uint(2), uint(1)).Return(&roomuser.RoomUser{Role: role.Role{Name: role.RoleListener}}, nil)
		authz := NewAuthz(ru)

		err := authz.CanCreate(&CreateChatMessageParams{RoomID: 1, UserID: 2, Content: "hi", IsPinned: true})

		require.ErrorIs(t, err, httpx.ErrForbidden)
		ru.AssertExpectations(t)
	})

	t.Run("failure: non-member is forbidden", func(t *testing.T) {
		ru := new(mockRoomUserService)
		ru.On("FindBy", uint(2), uint(1)).Return(nil, nil)
		authz := NewAuthz(ru)

		err := authz.CanCreate(&CreateChatMessageParams{RoomID: 1, UserID: 2, Content: "hi"})

		require.ErrorIs(t, err, httpx.ErrForbidden)
		ru.AssertExpectations(t)
	})
}
