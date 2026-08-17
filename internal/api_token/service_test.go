package apitoken

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"sound-stage-backend/internal/config"
	"sound-stage-backend/internal/pkg/httpx"
)

type mockRepository struct{ mock.Mock }

func (m *mockRepository) FindByToken(token string) (*APIToken, error) {
	args := m.Called(token)
	tok, _ := args.Get(0).(*APIToken)
	return tok, args.Error(1)
}
func (m *mockRepository) CreateToken(input CreateAPITokenInput) (*APIToken, error) {
	args := m.Called(input)
	tok, _ := args.Get(0).(*APIToken)
	return tok, args.Error(1)
}
func (m *mockRepository) Deactivate(userID uint) error {
	args := m.Called(userID)
	return args.Error(0)
}

func testConfig() *config.Config {
	return &config.Config{
		JWT: config.JWTConfig{
			AccessTokenSecret:  "test-access-secret",
			RefreshTokenSecret: "test-refresh-secret",
			AccessTokenExpiry:  15 * time.Minute,
			RefreshTokenExpiry: 7 * 24 * time.Hour,
			Issuer:             "soundstage-test",
		},
	}
}

type harness struct {
	repo *mockRepository
	svc  *Service
	cfg  *config.Config
}

func newHarness() *harness {
	cfg := testConfig()
	repo := new(mockRepository)
	return &harness{
		repo: repo,
		svc:  NewService(cfg, repo),
		cfg:  cfg,
	}
}

func newAPIToken(id, userID uint, token, typ string) *APIToken {
	t := &APIToken{Token: token, Type: typ, UserID: userID, IsActive: true}
	t.ID = id
	return t
}

func TestService_CreateToken(t *testing.T) {
	t.Run("success: signs a JWT and persists it via repo", func(t *testing.T) {
		h := newHarness()
		var capturedInput CreateAPITokenInput
		h.repo.On("CreateToken", mock.MatchedBy(func(in CreateAPITokenInput) bool {
			capturedInput = in
			return in.UserID == 12 && in.Type == AccessToken && in.Token != ""
		})).Return(newAPIToken(1, 12, "will-be-overwritten", "access"), nil)

		got, err := h.svc.CreateToken(12, AccessToken)

		require.NoError(t, err)
		require.NotNil(t, got)
		parsed, perr := jwt.Parse(capturedInput.Token, func(tok *jwt.Token) (any, error) {
			return []byte(h.cfg.JWT.AccessTokenSecret), nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
		require.NoError(t, perr)
		require.True(t, parsed.Valid)
		h.repo.AssertExpectations(t)
	})

	t.Run("failure: repo error is propagated", func(t *testing.T) {
		h := newHarness()
		repoErr := errors.New("insert failed")
		h.repo.On("CreateToken", mock.Anything).Return(nil, repoErr)

		got, err := h.svc.CreateToken(12, AccessToken)

		require.Nil(t, got)
		require.ErrorIs(t, err, repoErr)
		h.repo.AssertExpectations(t)
	})
}

func TestService_ValidateToken(t *testing.T) {
	t.Run("success: valid access token matching repo record returns user ID", func(t *testing.T) {
		h := newHarness()
		signed, err := h.svc.generateJWT(12, "access")
		require.NoError(t, err)

		h.repo.On("FindByToken", signed).Return(newAPIToken(1, 12, signed, "access"), nil)

		uid, err := h.svc.ValidateToken(signed, AccessToken)

		require.NoError(t, err)
		assert.Equal(t, uint(12), uid)
		h.repo.AssertExpectations(t)
	})

	t.Run("failure: token signed with wrong secret is rejected before hitting repo", func(t *testing.T) {
		h := newHarness()
		badlySigned, err := h.svc.generateJWT(12, "refresh")
		require.NoError(t, err)

		uid, err := h.svc.ValidateToken(badlySigned, AccessToken)

		require.ErrorIs(t, err, httpx.ErrInvalidOrExpiredToken)
		assert.Zero(t, uid)
		h.repo.AssertNotCalled(t, "FindByToken", mock.Anything)
	})
}

func TestService_Deactivate(t *testing.T) {
	t.Run("success: delegates to repo", func(t *testing.T) {
		h := newHarness()
		h.repo.On("Deactivate", uint(12)).Return(nil)

		err := h.svc.Deactivate(12)

		require.NoError(t, err)
		h.repo.AssertExpectations(t)
	})

	t.Run("failure: repo error is propagated", func(t *testing.T) {
		h := newHarness()
		repoErr := errors.New("update failed")
		h.repo.On("Deactivate", uint(12)).Return(repoErr)

		err := h.svc.Deactivate(12)

		require.ErrorIs(t, err, repoErr)
		h.repo.AssertExpectations(t)
	})
}
