package roomuserfavourite

import (
	"errors"
	"sound-stage-backend/internal/pkg/httpx"
	"sound-stage-backend/internal/pkg/listopts"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockRepo struct{ mock.Mock }

func (m *mockRepo) Add(userID, roomID uint) error {
	args := m.Called(userID, roomID)
	return args.Error(0)
}

func (m *mockRepo) Remove(userID, roomID uint) error {
	args := m.Called(userID, roomID)
	return args.Error(0)
}

func (m *mockRepo) ListUserFavourites(userID uint, p listopts.Pagination) ([]RoomUserFavourite, error) {
	args := m.Called(userID, p)
	favourites, _ := args.Get(0).([]RoomUserFavourite)
	return favourites, args.Error(1)
}

func (m *mockRepo) FindRoomIDsByUserID(userID uint, roomIDs []uint) ([]uint, error) {
	args := m.Called(userID, roomIDs)
	ids, _ := args.Get(0).([]uint)
	return ids, args.Error(1)
}

func (m *mockRepo) CountByUserID(userID uint) (int64, error) {
	args := m.Called(userID)
	return args.Get(0).(int64), args.Error(1)
}

type mockBlockedChecker struct{ mock.Mock }

func (m *mockBlockedChecker) IsBlocked(roomID, userID uint) (bool, error) {
	args := m.Called(roomID, userID)
	return args.Bool(0), args.Error(1)
}

type serviceHarness struct {
	repo    *mockRepo
	checker *mockBlockedChecker
	svc     *Service
}

func newServiceHarness() *serviceHarness {
	repo := new(mockRepo)
	checker := new(mockBlockedChecker)
	return &serviceHarness{
		repo:    repo,
		checker: checker,
		svc:     NewService(repo, NewAuthz(checker)),
	}
}

func (h *serviceHarness) assertAllExpectations(t *testing.T) {
	t.Helper()
	h.repo.AssertExpectations(t)
	h.checker.AssertExpectations(t)
}

func TestService_Add(t *testing.T) {
	t.Run("success: adds favourite through repo", func(t *testing.T) {
		h := newServiceHarness()
		h.checker.On("IsBlocked", uint(10), uint(20)).Return(false, nil)
		h.repo.On("Add", uint(20), uint(10)).Return(nil)

		err := h.svc.Add(20, 10)

		require.NoError(t, err)
		h.assertAllExpectations(t)
	})

	t.Run("failure: blocked user gets ErrUserBlocked before repo is called", func(t *testing.T) {
		h := newServiceHarness()
		h.checker.On("IsBlocked", uint(10), uint(20)).Return(true, nil)

		err := h.svc.Add(20, 10)

		require.ErrorIs(t, err, httpx.ErrUserBlocked)
		h.repo.AssertNotCalled(t, "Add", mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: IsBlocked error is propagated", func(t *testing.T) {
		h := newServiceHarness()
		blockErr := errors.New("block check failed")
		h.checker.On("IsBlocked", uint(10), uint(20)).Return(false, blockErr)

		err := h.svc.Add(20, 10)

		require.ErrorIs(t, err, blockErr)
		h.repo.AssertNotCalled(t, "Add", mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: repo error is propagated", func(t *testing.T) {
		h := newServiceHarness()
		h.checker.On("IsBlocked", uint(10), uint(20)).Return(false, nil)
		repoErr := errors.New("insert failed")
		h.repo.On("Add", uint(20), uint(10)).Return(repoErr)

		err := h.svc.Add(20, 10)

		require.ErrorIs(t, err, repoErr)
		h.assertAllExpectations(t)
	})
}

func TestService_Remove(t *testing.T) {
	t.Run("success: removes favourite through repo", func(t *testing.T) {
		h := newServiceHarness()
		h.repo.On("Remove", uint(20), uint(10)).Return(nil)

		err := h.svc.Remove(20, 10)

		require.NoError(t, err)
		h.assertAllExpectations(t)
	})

	t.Run("failure: repo error is propagated", func(t *testing.T) {
		h := newServiceHarness()
		repoErr := errors.New("delete failed")
		h.repo.On("Remove", uint(20), uint(10)).Return(repoErr)

		err := h.svc.Remove(20, 10)

		require.ErrorIs(t, err, repoErr)
		h.assertAllExpectations(t)
	})
}

func TestService_FavouritedRoomIDs(t *testing.T) {
	t.Run("success: returns set of favourited room ids", func(t *testing.T) {
		h := newServiceHarness()
		h.repo.On("FindRoomIDsByUserID", uint(20), []uint{10, 11}).Return([]uint{10}, nil)

		got, err := h.svc.FavouritedRoomIDs(20, []uint{10, 11})

		require.NoError(t, err)
		require.Equal(t, map[uint]bool{10: true}, got)
		h.assertAllExpectations(t)
	})

	t.Run("failure: repo error is propagated", func(t *testing.T) {
		h := newServiceHarness()
		repoErr := errors.New("query failed")
		h.repo.On("FindRoomIDsByUserID", uint(20), []uint{10}).Return(nil, repoErr)

		_, err := h.svc.FavouritedRoomIDs(20, []uint{10})

		require.ErrorIs(t, err, repoErr)
		h.assertAllExpectations(t)
	})
}

func TestService_ListUserFavourites(t *testing.T) {
	t.Run("success: returns paginated favourites", func(t *testing.T) {
		h := newServiceHarness()
		p := listopts.Pagination{Page: 1, PageSize: 10}
		favourites := []RoomUserFavourite{{UserID: 20, RoomID: 10}}
		h.repo.On("ListUserFavourites", uint(20), p).Return(favourites, nil)
		h.repo.On("CountByUserID", uint(20)).Return(int64(1), nil)

		got, count, err := h.svc.ListUserFavourites(20, p)

		require.NoError(t, err)
		require.Equal(t, favourites, got)
		require.Equal(t, int64(1), count)
		h.assertAllExpectations(t)
	})

	t.Run("failure: ListUserFavourites error is propagated", func(t *testing.T) {
		h := newServiceHarness()
		repoErr := errors.New("query failed")
		p := listopts.Pagination{Page: 1, PageSize: 10}
		h.repo.On("ListUserFavourites", uint(20), p).Return(nil, repoErr)

		_, _, err := h.svc.ListUserFavourites(20, p)

		require.ErrorIs(t, err, repoErr)
		h.repo.AssertNotCalled(t, "CountByUserID", mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: CountByUserID error is propagated", func(t *testing.T) {
		h := newServiceHarness()
		countErr := errors.New("count failed")
		p := listopts.Pagination{Page: 1, PageSize: 10}
		h.repo.On("ListUserFavourites", uint(20), p).Return([]RoomUserFavourite{}, nil)
		h.repo.On("CountByUserID", uint(20)).Return(int64(0), countErr)

		_, _, err := h.svc.ListUserFavourites(20, p)

		require.ErrorIs(t, err, countErr)
		h.assertAllExpectations(t)
	})
}
