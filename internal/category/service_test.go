package category

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockRepository struct{ mock.Mock }

func (m *mockRepository) List() ([]Category, error) {
	args := m.Called()
	categories, _ := args.Get(0).([]Category)
	return categories, args.Error(1)
}

func (m *mockRepository) Count() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

var _ repository = (*mockRepository)(nil)

type harness struct {
	repo *mockRepository
	svc  *Service
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	repo := new(mockRepository)
	return &harness{repo: repo, svc: NewService(repo)}
}

func TestService_List(t *testing.T) {
	t.Run("success: returns categories", func(t *testing.T) {
		h := newHarness(t)
		want := []Category{{Name: "Music"}}
		h.repo.On("List").Return(want, nil)

		got, err := h.svc.List()

		require.NoError(t, err)
		assert.Equal(t, want, got)
		h.repo.AssertExpectations(t)
	})

	t.Run("failure: List error propagated", func(t *testing.T) {
		h := newHarness(t)
		listErr := errors.New("list query failed")
		h.repo.On("List").Return(nil, listErr)

		got, err := h.svc.List()

		require.Nil(t, got)
		require.ErrorIs(t, err, listErr)
		h.repo.AssertExpectations(t)
	})
}
