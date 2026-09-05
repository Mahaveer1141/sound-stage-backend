package roomuserblock

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"sound-stage-backend/internal/pkg/httpx"
	"sound-stage-backend/internal/pkg/listopts"
	"sound-stage-backend/internal/role"
	roomuser "sound-stage-backend/internal/room_user"
)

type mockRepo struct{ mock.Mock }

func (m *mockRepo) Add(roomID, userID, blockedByID uint) error {
	args := m.Called(roomID, userID, blockedByID)
	return args.Error(0)
}

func (m *mockRepo) Remove(roomID, userID uint) error {
	args := m.Called(roomID, userID)
	return args.Error(0)
}
func (m *mockRepo) ListByRoomID(roomID uint, p listopts.Pagination) ([]RoomUserBlock, error) {
	args := m.Called(roomID, p)
	blocks, _ := args.Get(0).([]RoomUserBlock)
	return blocks, args.Error(1)
}
func (m *mockRepo) CountByRoomID(roomID uint) (int64, error) {
	args := m.Called(roomID)
	return args.Get(0).(int64), args.Error(1)
}

type mockRoomUserService struct{ mock.Mock }

func (m *mockRoomUserService) FindBy(userID, roomID uint) (*roomuser.RoomUser, error) {
	args := m.Called(userID, roomID)
	ru, _ := args.Get(0).(*roomuser.RoomUser)
	return ru, args.Error(1)
}

func (m *mockRoomUserService) HasRoles(userID, roomID uint, roles []role.RoleName) (bool, error) {
	args := m.Called(userID, roomID, roles)
	return args.Bool(0), args.Error(1)
}

type serviceHarness struct {
	repo        *mockRepo
	roomUserSvc *mockRoomUserService
	svc         *Service
}

func newServiceHarness() *serviceHarness {
	repo := new(mockRepo)
	roomUserSvc := new(mockRoomUserService)
	return &serviceHarness{
		repo:        repo,
		roomUserSvc: roomUserSvc,
		svc:         NewService(repo, roomUserSvc),
	}
}

func (h *serviceHarness) assertAllExpectations(t *testing.T) {
	t.Helper()
	h.repo.AssertExpectations(t)
	h.roomUserSvc.AssertExpectations(t)
}

func roomUser(userID, roomID uint, rn role.RoleName) *roomuser.RoomUser {
	return &roomuser.RoomUser{
		UserID: userID,
		RoomID: roomID,
		Role:   role.Role{Name: rn},
	}
}

func TestService_Add(t *testing.T) {
	t.Run("success: owner can block an admin", func(t *testing.T) {
		h := newServiceHarness()
		h.roomUserSvc.On("FindBy", uint(30), uint(10)).Return(roomUser(30, 10, role.RoleOwner), nil)
		h.roomUserSvc.On("FindBy", uint(20), uint(10)).Return(roomUser(20, 10, role.RoleAdmin), nil)
		h.repo.On("Add", uint(10), uint(20), uint(30)).Return(nil)

		err := h.svc.Add(10, 20, 30)

		require.NoError(t, err)
		h.assertAllExpectations(t)
	})

	t.Run("success: admin can block a listener", func(t *testing.T) {
		h := newServiceHarness()
		h.roomUserSvc.On("FindBy", uint(30), uint(10)).Return(roomUser(30, 10, role.RoleAdmin), nil)
		h.roomUserSvc.On("FindBy", uint(20), uint(10)).Return(roomUser(20, 10, role.RoleListener), nil)
		h.repo.On("Add", uint(10), uint(20), uint(30)).Return(nil)

		err := h.svc.Add(10, 20, 30)

		require.NoError(t, err)
		h.assertAllExpectations(t)
	})

	t.Run("success: moderator can block a speaker", func(t *testing.T) {
		h := newServiceHarness()
		h.roomUserSvc.On("FindBy", uint(30), uint(10)).Return(roomUser(30, 10, role.RoleModerator), nil)
		h.roomUserSvc.On("FindBy", uint(20), uint(10)).Return(roomUser(20, 10, role.RoleSpeaker), nil)
		h.repo.On("Add", uint(10), uint(20), uint(30)).Return(nil)

		err := h.svc.Add(10, 20, 30)

		require.NoError(t, err)
		h.assertAllExpectations(t)
	})

	t.Run("failure: admin cannot block an owner", func(t *testing.T) {
		h := newServiceHarness()
		h.roomUserSvc.On("FindBy", uint(30), uint(10)).Return(roomUser(30, 10, role.RoleAdmin), nil)
		h.roomUserSvc.On("FindBy", uint(20), uint(10)).Return(roomUser(20, 10, role.RoleOwner), nil)

		err := h.svc.Add(10, 20, 30)

		require.ErrorIs(t, err, httpx.ErrForbidden)
		h.repo.AssertNotCalled(t, "Add", mock.Anything, mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: moderator cannot block an admin", func(t *testing.T) {
		h := newServiceHarness()
		h.roomUserSvc.On("FindBy", uint(30), uint(10)).Return(roomUser(30, 10, role.RoleModerator), nil)
		h.roomUserSvc.On("FindBy", uint(20), uint(10)).Return(roomUser(20, 10, role.RoleAdmin), nil)

		err := h.svc.Add(10, 20, 30)

		require.ErrorIs(t, err, httpx.ErrForbidden)
		h.repo.AssertNotCalled(t, "Add", mock.Anything, mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: speaker cannot block anyone", func(t *testing.T) {
		h := newServiceHarness()
		h.roomUserSvc.On("FindBy", uint(30), uint(10)).Return(roomUser(30, 10, role.RoleSpeaker), nil)
		h.roomUserSvc.On("FindBy", uint(20), uint(10)).Return(roomUser(20, 10, role.RoleListener), nil)

		err := h.svc.Add(10, 20, 30)

		require.ErrorIs(t, err, httpx.ErrForbidden)
		h.repo.AssertNotCalled(t, "Add", mock.Anything, mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: listener cannot block anyone", func(t *testing.T) {
		h := newServiceHarness()
		h.roomUserSvc.On("FindBy", uint(30), uint(10)).Return(roomUser(30, 10, role.RoleListener), nil)
		h.roomUserSvc.On("FindBy", uint(20), uint(10)).Return(roomUser(20, 10, role.RoleListener), nil)

		err := h.svc.Add(10, 20, 30)

		require.ErrorIs(t, err, httpx.ErrForbidden)
		h.repo.AssertNotCalled(t, "Add", mock.Anything, mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: blocker who is not a room member is forbidden", func(t *testing.T) {
		h := newServiceHarness()
		h.roomUserSvc.On("FindBy", uint(30), uint(10)).Return(nil, nil)

		err := h.svc.Add(10, 20, 30)

		require.ErrorIs(t, err, httpx.ErrForbidden)
		h.roomUserSvc.AssertNotCalled(t, "FindBy", uint(20), uint(10))
		h.repo.AssertNotCalled(t, "Add", mock.Anything, mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: FindBy error is propagated", func(t *testing.T) {
		h := newServiceHarness()
		findErr := errors.New("db down")
		h.roomUserSvc.On("FindBy", uint(30), uint(10)).Return(nil, findErr)

		err := h.svc.Add(10, 20, 30)

		require.ErrorIs(t, err, findErr)
		h.repo.AssertNotCalled(t, "Add", mock.Anything, mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: repo.Add error is propagated", func(t *testing.T) {
		h := newServiceHarness()
		repoErr := errors.New("insert failed")
		h.roomUserSvc.On("FindBy", uint(30), uint(10)).Return(roomUser(30, 10, role.RoleOwner), nil)
		h.roomUserSvc.On("FindBy", uint(20), uint(10)).Return(roomUser(20, 10, role.RoleListener), nil)
		h.repo.On("Add", uint(10), uint(20), uint(30)).Return(repoErr)

		err := h.svc.Add(10, 20, 30)

		require.ErrorIs(t, err, repoErr)
		h.assertAllExpectations(t)
	})
}

func TestService_Remove(t *testing.T) {
	t.Run("success: owner can unblock an admin", func(t *testing.T) {
		h := newServiceHarness()
		h.roomUserSvc.On("FindBy", uint(30), uint(10)).Return(roomUser(30, 10, role.RoleOwner), nil)
		h.roomUserSvc.On("FindBy", uint(20), uint(10)).Return(roomUser(20, 10, role.RoleAdmin), nil)
		h.repo.On("Remove", uint(10), uint(20)).Return(nil)

		err := h.svc.Remove(10, 20, 30)

		require.NoError(t, err)
		h.assertAllExpectations(t)
	})

	t.Run("failure: admin cannot unblock an owner", func(t *testing.T) {
		h := newServiceHarness()
		h.roomUserSvc.On("FindBy", uint(30), uint(10)).Return(roomUser(30, 10, role.RoleAdmin), nil)
		h.roomUserSvc.On("FindBy", uint(20), uint(10)).Return(roomUser(20, 10, role.RoleOwner), nil)

		err := h.svc.Remove(10, 20, 30)

		require.ErrorIs(t, err, httpx.ErrForbidden)
		h.repo.AssertNotCalled(t, "Remove", mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: speaker cannot unblock anyone", func(t *testing.T) {
		h := newServiceHarness()
		h.roomUserSvc.On("FindBy", uint(30), uint(10)).Return(roomUser(30, 10, role.RoleSpeaker), nil)
		h.roomUserSvc.On("FindBy", uint(20), uint(10)).Return(roomUser(20, 10, role.RoleListener), nil)

		err := h.svc.Remove(10, 20, 30)

		require.ErrorIs(t, err, httpx.ErrForbidden)
		h.repo.AssertNotCalled(t, "Remove", mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: repo.Remove error is propagated", func(t *testing.T) {
		h := newServiceHarness()
		repoErr := errors.New("delete failed")
		h.roomUserSvc.On("FindBy", uint(30), uint(10)).Return(roomUser(30, 10, role.RoleOwner), nil)
		h.roomUserSvc.On("FindBy", uint(20), uint(10)).Return(roomUser(20, 10, role.RoleListener), nil)
		h.repo.On("Remove", uint(10), uint(20)).Return(repoErr)

		err := h.svc.Remove(10, 20, 30)

		require.ErrorIs(t, err, repoErr)
		h.assertAllExpectations(t)
	})
}

func TestService_ListByRoomID(t *testing.T) {
	adminRoles := []role.RoleName{role.RoleAdmin, role.RoleOwner}

	t.Run("success: owner lists blocked users with pagination", func(t *testing.T) {
		h := newServiceHarness()
		h.roomUserSvc.On("HasRoles", uint(30), uint(10), adminRoles).Return(true, nil)
		p := listopts.Pagination{Page: 1, PageSize: 10}
		blocks := []RoomUserBlock{{UserID: 20, RoomID: 10, BlockedByID: 30}}
		h.repo.On("ListByRoomID", uint(10), p).Return(blocks, nil)
		h.repo.On("CountByRoomID", uint(10)).Return(int64(1), nil)

		got, count, err := h.svc.ListByRoomID(10, 30, p)

		require.NoError(t, err)
		require.Equal(t, blocks, got)
		require.Equal(t, int64(1), count)
		h.assertAllExpectations(t)
	})

	t.Run("success: admin lists blocked users", func(t *testing.T) {
		h := newServiceHarness()
		h.roomUserSvc.On("HasRoles", uint(30), uint(10), adminRoles).Return(true, nil)
		p := listopts.Pagination{Page: 1, PageSize: 10}
		h.repo.On("ListByRoomID", uint(10), p).Return([]RoomUserBlock{}, nil)
		h.repo.On("CountByRoomID", uint(10)).Return(int64(0), nil)

		_, count, err := h.svc.ListByRoomID(10, 30, p)

		require.NoError(t, err)
		require.Equal(t, int64(0), count)
		h.assertAllExpectations(t)
	})

	t.Run("failure: moderator cannot list blocked users", func(t *testing.T) {
		h := newServiceHarness()
		h.roomUserSvc.On("HasRoles", uint(30), uint(10), adminRoles).Return(false, nil)

		_, _, err := h.svc.ListByRoomID(10, 30, listopts.Pagination{Page: 1, PageSize: 10})

		require.ErrorIs(t, err, httpx.ErrForbidden)
		h.repo.AssertNotCalled(t, "ListByRoomID", mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: non-member cannot list blocked users", func(t *testing.T) {
		h := newServiceHarness()
		h.roomUserSvc.On("HasRoles", uint(30), uint(10), adminRoles).Return(false, nil)

		_, _, err := h.svc.ListByRoomID(10, 30, listopts.Pagination{Page: 1, PageSize: 10})

		require.ErrorIs(t, err, httpx.ErrForbidden)
		h.repo.AssertNotCalled(t, "ListByRoomID", mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: HasRoles error is propagated", func(t *testing.T) {
		h := newServiceHarness()
		permErr := errors.New("permission check failed")
		h.roomUserSvc.On("HasRoles", uint(30), uint(10), adminRoles).Return(false, permErr)

		_, _, err := h.svc.ListByRoomID(10, 30, listopts.Pagination{Page: 1, PageSize: 10})

		require.ErrorIs(t, err, permErr)
		h.repo.AssertNotCalled(t, "ListByRoomID", mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: ListByRoomID error is propagated", func(t *testing.T) {
		h := newServiceHarness()
		repoErr := errors.New("query failed")
		h.roomUserSvc.On("HasRoles", uint(30), uint(10), adminRoles).Return(true, nil)
		p := listopts.Pagination{Page: 1, PageSize: 10}
		h.repo.On("ListByRoomID", uint(10), p).Return(nil, repoErr)

		_, _, err := h.svc.ListByRoomID(10, 30, p)

		require.ErrorIs(t, err, repoErr)
		h.repo.AssertNotCalled(t, "CountByRoomID", mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: CountByRoomID error is propagated", func(t *testing.T) {
		h := newServiceHarness()
		countErr := errors.New("count failed")
		h.roomUserSvc.On("HasRoles", uint(30), uint(10), adminRoles).Return(true, nil)
		p := listopts.Pagination{Page: 1, PageSize: 10}
		h.repo.On("ListByRoomID", uint(10), p).Return([]RoomUserBlock{}, nil)
		h.repo.On("CountByRoomID", uint(10)).Return(int64(0), countErr)

		_, _, err := h.svc.ListByRoomID(10, 30, p)

		require.ErrorIs(t, err, countErr)
		h.assertAllExpectations(t)
	})
}
