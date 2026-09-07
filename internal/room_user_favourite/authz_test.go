package roomuserfavourite

import (
	"errors"
	"sound-stage-backend/internal/pkg/httpx"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAuthz_CanAdd(t *testing.T) {
	t.Run("success: not blocked", func(t *testing.T) {
		checker := new(mockBlockedChecker)
		checker.On("IsBlocked", uint(10), uint(20)).Return(false, nil)
		authz := NewAuthz(checker)

		err := authz.CanAdd(20, 10)

		require.NoError(t, err)
		checker.AssertExpectations(t)
	})

	t.Run("failure: blocked user gets ErrUserBlocked", func(t *testing.T) {
		checker := new(mockBlockedChecker)
		checker.On("IsBlocked", uint(10), uint(20)).Return(true, nil)
		authz := NewAuthz(checker)

		err := authz.CanAdd(20, 10)

		require.ErrorIs(t, err, httpx.ErrUserBlocked)
		checker.AssertExpectations(t)
	})

	t.Run("failure: IsBlocked error is propagated", func(t *testing.T) {
		checker := new(mockBlockedChecker)
		checkErr := errors.New("block check failed")
		checker.On("IsBlocked", uint(10), uint(20)).Return(false, checkErr)
		authz := NewAuthz(checker)

		err := authz.CanAdd(20, 10)

		require.ErrorIs(t, err, checkErr)
		checker.AssertExpectations(t)
	})
}
