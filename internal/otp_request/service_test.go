package otprequest

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockRepository struct{ mock.Mock }

func (m *mockRepository) FindByEmail(email string) (*OTPRequest, error) {
	args := m.Called(email)
	o, _ := args.Get(0).(*OTPRequest)
	return o, args.Error(1)
}
func (m *mockRepository) Create(input CreateOTPRequestInput) (*OTPRequest, error) {
	args := m.Called(input)
	o, _ := args.Get(0).(*OTPRequest)
	return o, args.Error(1)
}
func (m *mockRepository) Deactivate(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func TestService_FindByEmail(t *testing.T) {
	t.Run("success: returns the exact record from repo", func(t *testing.T) {
		repo := new(mockRepository)
		want := &OTPRequest{OTP: "123456"}
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
		input := CreateOTPRequestInput{OTP: "654321"}
		want := &OTPRequest{OTP: "654321"}
		repo.On("Create", input).Return(want, nil)

		svc := NewService(repo)
		got, err := svc.Create(input)

		require.NoError(t, err)
		assert.Same(t, want, got)
		repo.AssertExpectations(t)
	})

	t.Run("failure: repo error is propagated unchanged", func(t *testing.T) {
		repo := new(mockRepository)
		input := CreateOTPRequestInput{OTP: "654321"}
		repoErr := errors.New("duplicate active request")
		repo.On("Create", input).Return(nil, repoErr)

		svc := NewService(repo)
		got, err := svc.Create(input)

		require.Nil(t, got)
		require.ErrorIs(t, err, repoErr)
		repo.AssertExpectations(t)
	})
}

func TestService_Deactivate(t *testing.T) {
	t.Run("success: delegates to repo", func(t *testing.T) {
		repo := new(mockRepository)
		repo.On("Deactivate", uint(9)).Return(nil)

		svc := NewService(repo)
		err := svc.Deactivate(9)

		require.NoError(t, err)
		repo.AssertExpectations(t)
	})

	t.Run("failure: repo error is propagated unchanged", func(t *testing.T) {
		repo := new(mockRepository)
		repoErr := errors.New("update failed")
		repo.On("Deactivate", uint(9)).Return(repoErr)

		svc := NewService(repo)
		err := svc.Deactivate(9)

		require.ErrorIs(t, err, repoErr)
		repo.AssertExpectations(t)
	})
}
