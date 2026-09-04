package roomuserfavourite

import (
	"errors"
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

type serviceHarness struct {
	repo *mockRepo
	svc  *Service
}

func newServiceHarness() *serviceHarness {
	repo := new(mockRepo)
	return &serviceHarness{
		repo: repo,
		svc:  NewService(repo),
	}
}

func (h *serviceHarness) assertAllExpectations(t *testing.T) {
	t.Helper()
	h.repo.AssertExpectations(t)
}

func TestService_Add(t *testing.T) {
	t.Run("success: adds favourite through repo", func(t *testing.T) {
		h := newServiceHarness()
		h.repo.On("Add", uint(20), uint(10)).Return(nil)

		err := h.svc.Add(20, 10)

		require.NoError(t, err)
		h.assertAllExpectations(t)
	})

	t.Run("failure: repo error is propagated", func(t *testing.T) {
		h := newServiceHarness()
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
