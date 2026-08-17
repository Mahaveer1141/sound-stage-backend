package user

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockRepository struct{ mock.Mock }

func (m *mockRepository) Create(input *CreateUserParams) (*User, error) {
	args := m.Called(input)
	u, _ := args.Get(0).(*User)
	return u, args.Error(1)
}
func (m *mockRepository) FindByID(id uint) (*User, error) {
	args := m.Called(id)
	u, _ := args.Get(0).(*User)
	return u, args.Error(1)
}
func (m *mockRepository) FindByEmail(email string) (*User, error) {
	args := m.Called(email)
	u, _ := args.Get(0).(*User)
	return u, args.Error(1)
}
func (m *mockRepository) UpdateLastLoginAt(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}
func (m *mockRepository) Update(id uint, input *UpdateUserParams) (*User, error) {
	args := m.Called(id, input)
	u, _ := args.Get(0).(*User)
	return u, args.Error(1)
}

func TestService_FindByID(t *testing.T) {
	t.Run("success: returns the exact record from repo", func(t *testing.T) {
		repo := new(mockRepository)
		want := &User{Email: "a@example.com"}
		repo.On("FindByID", uint(1)).Return(want, nil)

		svc := NewService(repo)
		got, err := svc.FindByID(1)

		require.NoError(t, err)
		assert.Same(t, want, got)
		repo.AssertExpectations(t)
	})

	t.Run("failure: repo error is propagated unchanged", func(t *testing.T) {
		repo := new(mockRepository)
		repoErr := errors.New("not found")
		repo.On("FindByID", uint(99)).Return(nil, repoErr)

		svc := NewService(repo)
		got, err := svc.FindByID(99)

		require.Nil(t, got)
		require.ErrorIs(t, err, repoErr)
		repo.AssertExpectations(t)
	})
}

func TestService_FindByEmail(t *testing.T) {
	t.Run("success: returns the exact record from repo", func(t *testing.T) {
		repo := new(mockRepository)
		want := &User{Email: "a@example.com"}
		repo.On("FindByEmail", "a@example.com").Return(want, nil)

		svc := NewService(repo)
		got, err := svc.FindByEmail("a@example.com")

		require.NoError(t, err)
		assert.Same(t, want, got)
		repo.AssertExpectations(t)
	})

	t.Run("failure: repo error is propagated unchanged", func(t *testing.T) {
		repo := new(mockRepository)
		repoErr := errors.New("not found")
		repo.On("FindByEmail", "missing@example.com").Return(nil, repoErr)

		svc := NewService(repo)
		got, err := svc.FindByEmail("missing@example.com")

		require.Nil(t, got)
		require.ErrorIs(t, err, repoErr)
		repo.AssertExpectations(t)
	})
}

func TestService_Create(t *testing.T) {
	t.Run("success: returns the created record from repo", func(t *testing.T) {
		repo := new(mockRepository)
		input := &CreateUserParams{Email: "new@example.com", FirstName: "Ada"}
		want := &User{Email: "new@example.com", FirstName: "Ada"}
		repo.On("Create", input).Return(want, nil)

		svc := NewService(repo)
		got, err := svc.Create(input)

		require.NoError(t, err)
		assert.Same(t, want, got)
		repo.AssertExpectations(t)
	})

	t.Run("failure: repo error is propagated unchanged (e.g. duplicate email)", func(t *testing.T) {
		repo := new(mockRepository)
		input := &CreateUserParams{Email: "dup@example.com", FirstName: "Ada"}
		repoErr := errors.New("duplicate email")
		repo.On("Create", input).Return(nil, repoErr)

		svc := NewService(repo)
		got, err := svc.Create(input)

		require.Nil(t, got)
		require.ErrorIs(t, err, repoErr)
		repo.AssertExpectations(t)
	})
}

func TestService_UpdateLastLoginAt(t *testing.T) {
	t.Run("success: delegates to repo", func(t *testing.T) {
		repo := new(mockRepository)
		repo.On("UpdateLastLoginAt", uint(1)).Return(nil)

		svc := NewService(repo)
		err := svc.UpdateLastLoginAt(1)

		require.NoError(t, err)
		repo.AssertExpectations(t)
	})

	t.Run("failure: repo error is propagated unchanged", func(t *testing.T) {
		repo := new(mockRepository)
		repoErr := errors.New("update failed")
		repo.On("UpdateLastLoginAt", uint(1)).Return(repoErr)

		svc := NewService(repo)
		err := svc.UpdateLastLoginAt(1)

		require.ErrorIs(t, err, repoErr)
		repo.AssertExpectations(t)
	})
}

func TestService_UpdateProfile(t *testing.T) {
	t.Run("success: returns the updated record from repo", func(t *testing.T) {
		repo := new(mockRepository)
		input := &UpdateUserParams{FirstName: "Grace"}
		want := &User{FirstName: "Grace"}
		repo.On("Update", uint(1), input).Return(want, nil)

		svc := NewService(repo)
		got, err := svc.UpdateProfile(1, input)

		require.NoError(t, err)
		assert.Same(t, want, got)
		repo.AssertExpectations(t)
	})

	t.Run("failure: repo error is propagated unchanged (e.g. user not found)", func(t *testing.T) {
		repo := new(mockRepository)
		input := &UpdateUserParams{FirstName: "Grace"}
		repoErr := errors.New("user not found")
		repo.On("Update", uint(99), input).Return(nil, repoErr)

		svc := NewService(repo)
		got, err := svc.UpdateProfile(99, input)

		require.Nil(t, got)
		require.ErrorIs(t, err, repoErr)
		repo.AssertExpectations(t)
	})
}
