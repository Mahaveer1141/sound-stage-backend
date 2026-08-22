package tag

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"sound-stage-backend/internal/pkg/listopts"
)

type mockRepository struct{ mock.Mock }

func (m *mockRepository) Create(input *CreateTagParams) (*Tag, error) {
	args := m.Called(input)
	tag, _ := args.Get(0).(*Tag)
	return tag, args.Error(1)
}
func (m *mockRepository) List(filter TagFilter, sort listopts.Sort, p listopts.Pagination) ([]Tag, error) {
	args := m.Called(filter, sort, p)
	tags, _ := args.Get(0).([]Tag)
	return tags, args.Error(1)
}
func (m *mockRepository) Count(filter TagFilter) (int64, error) {
	args := m.Called(filter)
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

func TestService_Create(t *testing.T) {
	t.Run("creates a new tag when name is not taken", func(t *testing.T) {
		h := newHarness(t)
		input := &CreateTagParams{Name: "live"}
		want := &Tag{Name: "live"}
		h.repo.On("Create", input).Return(want, nil)

		got, err := h.svc.Create(input)

		require.NoError(t, err)
		assert.Same(t, want, got)
		h.repo.AssertExpectations(t)
	})

	t.Run("failure: Create error propagated", func(t *testing.T) {
		h := newHarness(t)
		input := &CreateTagParams{Name: "live"}
		createErr := errors.New("insert failed")
		h.repo.On("Create", input).Return(nil, createErr)

		got, err := h.svc.Create(input)

		require.Nil(t, got)
		require.ErrorIs(t, err, createErr)
		h.repo.AssertExpectations(t)
	})
}
func TestService_List(t *testing.T) {
	t.Run("success: returns tags and count", func(t *testing.T) {
		h := newHarness(t)
		filter := TagFilter{}
		sort := listopts.Sort{}
		p := listopts.Pagination{Page: 1, PageSize: 10}
		want := []Tag{{Name: "live"}}
		h.repo.On("List", filter, sort, p).Return(want, nil)
		h.repo.On("Count", filter).Return(int64(1), nil)

		got, count, err := h.svc.List(filter, sort, p)

		require.NoError(t, err)
		assert.Equal(t, want, got)
		assert.Equal(t, int64(1), count)
		h.repo.AssertExpectations(t)
	})

	t.Run("failure: List error propagated", func(t *testing.T) {
		h := newHarness(t)
		filter := TagFilter{}
		sort := listopts.Sort{}
		p := listopts.Pagination{Page: 1, PageSize: 10}
		listErr := errors.New("list query failed")
		h.repo.On("List", filter, sort, p).Return(nil, listErr)

		got, count, err := h.svc.List(filter, sort, p)

		require.Nil(t, got)
		require.Zero(t, count)
		require.ErrorIs(t, err, listErr)
		h.repo.AssertNotCalled(t, "Count", mock.Anything)
		h.repo.AssertExpectations(t)
	})

	t.Run("failure: Count error after successful List still fails the call", func(t *testing.T) {
		h := newHarness(t)
		filter := TagFilter{}
		sort := listopts.Sort{}
		p := listopts.Pagination{Page: 1, PageSize: 10}
		tags := []Tag{{Name: "live"}}
		countErr := errors.New("count query failed")
		h.repo.On("List", filter, sort, p).Return(tags, nil)
		h.repo.On("Count", filter).Return(int64(0), countErr)

		got, count, err := h.svc.List(filter, sort, p)

		require.Nil(t, got)
		require.Zero(t, count)
		require.ErrorIs(t, err, countErr)
		h.repo.AssertExpectations(t)
	})

	t.Run("success: passes search query through to repo", func(t *testing.T) {
		h := newHarness(t)
		filter := TagFilter{Query: "li"}
		sort := listopts.Sort{Field: "name", Order: "asc"}
		p := listopts.Pagination{Page: 1, PageSize: 10}
		want := []Tag{{Name: "live"}}
		h.repo.On("List", filter, sort, p).Return(want, nil)
		h.repo.On("Count", filter).Return(int64(1), nil)

		got, count, err := h.svc.List(filter, sort, p)

		require.NoError(t, err)
		assert.Equal(t, want, got)
		assert.Equal(t, int64(1), count)
		h.repo.AssertExpectations(t)
	})
}
