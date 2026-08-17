package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	apitoken "sound-stage-backend/internal/api_token"
	otprequest "sound-stage-backend/internal/otp_request"
	"sound-stage-backend/internal/pkg/httpx"
	"sound-stage-backend/internal/user"
)

type mockUserService struct{ mock.Mock }

func (m *mockUserService) FindByEmail(email string) (*user.User, error) {
	args := m.Called(email)
	u, _ := args.Get(0).(*user.User)
	return u, args.Error(1)
}
func (m *mockUserService) Create(input *user.CreateUserParams) (*user.User, error) {
	args := m.Called(input)
	u, _ := args.Get(0).(*user.User)
	return u, args.Error(1)
}
func (m *mockUserService) UpdateLastLoginAt(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

type mockOTPService struct{ mock.Mock }

func (m *mockOTPService) FindByEmail(email string) (*otprequest.OTPRequest, error) {
	args := m.Called(email)
	o, _ := args.Get(0).(*otprequest.OTPRequest)
	return o, args.Error(1)
}
func (m *mockOTPService) Create(input otprequest.CreateOTPRequestInput) (*otprequest.OTPRequest, error) {
	args := m.Called(input)
	o, _ := args.Get(0).(*otprequest.OTPRequest)
	return o, args.Error(1)
}
func (m *mockOTPService) Deactivate(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

type mockTokenManager struct{ mock.Mock }

func (m *mockTokenManager) CreateToken(userID uint, tokenType apitoken.TokenType) (*apitoken.APIToken, error) {
	args := m.Called(userID, tokenType)
	tok, _ := args.Get(0).(*apitoken.APIToken)
	return tok, args.Error(1)
}
func (m *mockTokenManager) ValidateToken(token string, tokenType apitoken.TokenType) (uint, error) {
	args := m.Called(token, tokenType)
	return args.Get(0).(uint), args.Error(1)
}
func (m *mockTokenManager) Deactivate(userID uint) error {
	args := m.Called(userID)
	return args.Error(0)
}

type mockMailer struct{ mock.Mock }

func (m *mockMailer) SendOTPEmail(ctx context.Context, to string, data map[string]any) {
	m.Called(ctx, to, data)
}

type harness struct {
	users  *mockUserService
	otps   *mockOTPService
	tokens *mockTokenManager
	mail   *mockMailer
	svc    *Service
}

func newHarness() *harness {
	h := &harness{
		users:  new(mockUserService),
		otps:   new(mockOTPService),
		tokens: new(mockTokenManager),
		mail:   new(mockMailer),
	}
	h.svc = NewService(h.users, h.otps, h.tokens, h.mail)
	return h
}

func (h *harness) assertAllExpectations(t *testing.T) {
	t.Helper()
	h.users.AssertExpectations(t)
	h.otps.AssertExpectations(t)
	h.tokens.AssertExpectations(t)
	h.mail.AssertExpectations(t)
}

func futureTime() time.Time { return time.Now().Add(time.Hour) }
func pastTime() time.Time   { return time.Now().Add(-time.Hour) }

func newUser(id uint) *user.User {
	u := &user.User{Email: "placeholder@example.com", FirstName: "Placeholder"}
	u.ID = id
	return u
}

func newAPIToken(id uint, tokenType string) *apitoken.APIToken {
	tok := &apitoken.APIToken{Type: tokenType, IsActive: true}
	tok.ID = id
	return tok
}

func TestService_RequestOTP(t *testing.T) {
	t.Run("existing user: sets UserID, leaves Email nil, sends mail", func(t *testing.T) {
		h := newHarness()
		existing := newUser(42)

		h.users.On("FindByEmail", "a@example.com").Return(existing, nil)
		h.otps.On("Create", mock.MatchedBy(func(in otprequest.CreateOTPRequestInput) bool {
			return in.UserID != nil && *in.UserID == 42 && in.Email == nil && len(in.OTP) == 6
		})).Return(&otprequest.OTPRequest{}, nil)
		h.mail.On("SendOTPEmail", mock.Anything, "a@example.com", mock.Anything).Return()

		got, err := h.svc.RequestOTP("a@example.com")

		require.NoError(t, err)
		require.NotNil(t, got)
		h.assertAllExpectations(t)
	})

	t.Run("unregistered email: sets Email, leaves UserID nil", func(t *testing.T) {
		h := newHarness()

		h.users.On("FindByEmail", "new@example.com").Return(nil, nil)
		h.otps.On("Create", mock.MatchedBy(func(in otprequest.CreateOTPRequestInput) bool {
			return in.UserID == nil && in.Email != nil && *in.Email == "new@example.com"
		})).Return(&otprequest.OTPRequest{}, nil)
		h.mail.On("SendOTPEmail", mock.Anything, "new@example.com", mock.Anything).Return()

		got, err := h.svc.RequestOTP("new@example.com")

		require.NoError(t, err)
		require.NotNil(t, got)
		h.assertAllExpectations(t)
	})

	t.Run("generated OTP is 6 numeric digits", func(t *testing.T) {
		h := newHarness()
		h.users.On("FindByEmail", "a@example.com").Return(nil, nil)

		var captured string
		h.otps.On("Create", mock.MatchedBy(func(in otprequest.CreateOTPRequestInput) bool {
			captured = in.OTP
			return true
		})).Return(&otprequest.OTPRequest{}, nil)
		h.mail.On("SendOTPEmail", mock.Anything, mock.Anything, mock.Anything).Return()

		_, err := h.svc.RequestOTP("a@example.com")

		require.NoError(t, err)
		require.Len(t, captured, 6)
		for _, c := range captured {
			assert.True(t, c >= '0' && c <= '9', "expected digit, got %q", c)
		}
	})

	t.Run("user lookup error short-circuits before OTP creation", func(t *testing.T) {
		h := newHarness()
		lookupErr := errors.New("db down")
		h.users.On("FindByEmail", "a@example.com").Return(nil, lookupErr)

		got, err := h.svc.RequestOTP("a@example.com")

		require.Nil(t, got)
		require.ErrorIs(t, err, lookupErr)
		h.otps.AssertNotCalled(t, "Create", mock.Anything)
		h.mail.AssertNotCalled(t, "SendOTPEmail", mock.Anything, mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("otp create error is returned and mail is never sent", func(t *testing.T) {
		h := newHarness()
		createErr := errors.New("write failed")
		h.users.On("FindByEmail", "a@example.com").Return(nil, nil)
		h.otps.On("Create", mock.Anything).Return(nil, createErr)

		got, err := h.svc.RequestOTP("a@example.com")

		require.Nil(t, got)
		require.ErrorIs(t, err, createErr)
		h.mail.AssertNotCalled(t, "SendOTPEmail", mock.Anything, mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})
}

func TestService_VerifyOTP(t *testing.T) {
	t.Run("valid OTP for registered user issues tokens and updates last login", func(t *testing.T) {
		h := newHarness()
		uid := uint(7)
		otpReq := otprequest.OTPRequest{UserID: &uid, OTP: "123456", IsActive: true, ExpiresAt: futureTime()}
		otpReq.ID = 100
		h.otps.On("FindByEmail", "a@example.com").Return(&otpReq, nil)
		h.otps.On("Deactivate", uint(100)).Return(nil)
		access := newAPIToken(201, "access")
		refresh := newAPIToken(202, "refresh")
		h.tokens.On("CreateToken", uid, apitoken.AccessToken).Return(access, nil)
		h.tokens.On("CreateToken", uid, apitoken.RefreshToken).Return(refresh, nil)
		h.users.On("UpdateLastLoginAt", uid).Return(nil)

		got, err := h.svc.VerifyOTP(VerifyOTPParams{Email: "a@example.com", OTP: "123456"})

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Same(t, access, got.AccessToken)
		assert.Same(t, refresh, got.RefreshToken)
		h.assertAllExpectations(t)
	})

	t.Run("unregistered email (nil UserID) deactivates OTP but issues no tokens", func(t *testing.T) {
		h := newHarness()
		otpReq := otprequest.OTPRequest{UserID: nil, OTP: "654321", IsActive: true, ExpiresAt: futureTime()}
		otpReq.ID = 101
		h.otps.On("FindByEmail", "new@example.com").Return(&otpReq, nil)
		h.otps.On("Deactivate", uint(101)).Return(nil)

		got, err := h.svc.VerifyOTP(VerifyOTPParams{Email: "new@example.com", OTP: "654321"})

		require.NoError(t, err)
		require.Nil(t, got)
		h.tokens.AssertNotCalled(t, "CreateToken", mock.Anything, mock.Anything)
		h.users.AssertNotCalled(t, "UpdateLastLoginAt", mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("wrong OTP returns ErrInvalidOTP and never deactivates", func(t *testing.T) {
		h := newHarness()
		uid := uint(7)
		otpReq := otprequest.OTPRequest{UserID: &uid, OTP: "123456", IsActive: true, ExpiresAt: futureTime()}
		otpReq.ID = 100
		h.otps.On("FindByEmail", "a@example.com").Return(&otpReq, nil)

		got, err := h.svc.VerifyOTP(VerifyOTPParams{Email: "a@example.com", OTP: "000000"})

		require.Nil(t, got)
		require.ErrorIs(t, err, httpx.ErrInvalidOTP)
		h.otps.AssertNotCalled(t, "Deactivate", mock.Anything)
		h.tokens.AssertNotCalled(t, "CreateToken", mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("expired OTP returns ErrInvalidOTP", func(t *testing.T) {
		h := newHarness()
		uid := uint(7)
		otpReq := otprequest.OTPRequest{UserID: &uid, OTP: "123456", IsActive: true, ExpiresAt: pastTime()}
		otpReq.ID = 100
		h.otps.On("FindByEmail", "a@example.com").Return(&otpReq, nil)

		got, err := h.svc.VerifyOTP(VerifyOTPParams{Email: "a@example.com", OTP: "123456"})

		require.Nil(t, got)
		require.ErrorIs(t, err, httpx.ErrInvalidOTP)
		h.otps.AssertNotCalled(t, "Deactivate", mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("inactive OTP returns ErrInvalidOTP", func(t *testing.T) {
		h := newHarness()
		uid := uint(7)
		otpReq := otprequest.OTPRequest{UserID: &uid, OTP: "123456", IsActive: false, ExpiresAt: futureTime()}
		otpReq.ID = 100
		h.otps.On("FindByEmail", "a@example.com").Return(&otpReq, nil)

		got, err := h.svc.VerifyOTP(VerifyOTPParams{Email: "a@example.com", OTP: "123456"})

		require.Nil(t, got)
		require.ErrorIs(t, err, httpx.ErrInvalidOTP)
		h.assertAllExpectations(t)
	})

	t.Run("otp lookup error is propagated", func(t *testing.T) {
		h := newHarness()
		lookupErr := errors.New("not found")
		h.otps.On("FindByEmail", "a@example.com").Return(nil, lookupErr)

		got, err := h.svc.VerifyOTP(VerifyOTPParams{Email: "a@example.com", OTP: "123456"})

		require.Nil(t, got)
		require.ErrorIs(t, err, lookupErr)
		h.assertAllExpectations(t)
	})

	t.Run("deactivate error is propagated and blocks token issuance", func(t *testing.T) {
		h := newHarness()
		uid := uint(7)
		otpReq := otprequest.OTPRequest{UserID: &uid, OTP: "123456", IsActive: true, ExpiresAt: futureTime()}
		otpReq.ID = 100
		h.otps.On("FindByEmail", "a@example.com").Return(&otpReq, nil)
		deactivateErr := errors.New("deactivate failed")
		h.otps.On("Deactivate", uint(100)).Return(deactivateErr)

		got, err := h.svc.VerifyOTP(VerifyOTPParams{Email: "a@example.com", OTP: "123456"})

		require.Nil(t, got)
		require.ErrorIs(t, err, deactivateErr)
		h.tokens.AssertNotCalled(t, "CreateToken", mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("access token creation failure aborts before refresh token / last login", func(t *testing.T) {
		h := newHarness()
		uid := uint(7)
		otpReq := otprequest.OTPRequest{UserID: &uid, OTP: "123456", IsActive: true, ExpiresAt: futureTime()}
		otpReq.ID = 100
		h.otps.On("FindByEmail", "a@example.com").Return(&otpReq, nil)
		h.otps.On("Deactivate", uint(100)).Return(nil)
		tokenErr := errors.New("token creation failed")
		h.tokens.On("CreateToken", uid, apitoken.AccessToken).Return(nil, tokenErr)

		got, err := h.svc.VerifyOTP(VerifyOTPParams{Email: "a@example.com", OTP: "123456"})

		require.Nil(t, got)
		require.ErrorIs(t, err, tokenErr)
		h.tokens.AssertNotCalled(t, "CreateToken", uid, apitoken.RefreshToken)
		h.users.AssertNotCalled(t, "UpdateLastLoginAt", mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("refresh token creation failure aborts before last-login update", func(t *testing.T) {
		h := newHarness()
		uid := uint(7)
		otpReq := otprequest.OTPRequest{UserID: &uid, OTP: "123456", IsActive: true, ExpiresAt: futureTime()}
		otpReq.ID = 100
		h.otps.On("FindByEmail", "a@example.com").Return(&otpReq, nil)
		h.otps.On("Deactivate", uint(100)).Return(nil)
		h.tokens.On("CreateToken", uid, apitoken.AccessToken).Return(newAPIToken(1, "access"), nil)
		tokenErr := errors.New("refresh token creation failed")
		h.tokens.On("CreateToken", uid, apitoken.RefreshToken).Return(nil, tokenErr)

		got, err := h.svc.VerifyOTP(VerifyOTPParams{Email: "a@example.com", OTP: "123456"})

		require.Nil(t, got)
		require.ErrorIs(t, err, tokenErr)
		h.users.AssertNotCalled(t, "UpdateLastLoginAt", mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("last-login update failure surfaces even though tokens were already created", func(t *testing.T) {
		h := newHarness()
		uid := uint(7)
		otpReq := otprequest.OTPRequest{UserID: &uid, OTP: "123456", IsActive: true, ExpiresAt: futureTime()}
		otpReq.ID = 100
		h.otps.On("FindByEmail", "a@example.com").Return(&otpReq, nil)
		h.otps.On("Deactivate", uint(100)).Return(nil)
		h.tokens.On("CreateToken", uid, apitoken.AccessToken).Return(newAPIToken(1, "access"), nil)
		h.tokens.On("CreateToken", uid, apitoken.RefreshToken).Return(newAPIToken(2, "refresh"), nil)
		loginErr := errors.New("update failed")
		h.users.On("UpdateLastLoginAt", uid).Return(loginErr)

		got, err := h.svc.VerifyOTP(VerifyOTPParams{Email: "a@example.com", OTP: "123456"})

		require.Nil(t, got)
		require.ErrorIs(t, err, loginErr)
		h.assertAllExpectations(t)
	})
}

func TestService_SignUp(t *testing.T) {
	t.Run("creates user and issues both tokens", func(t *testing.T) {
		h := newHarness()
		created := newUser(55)
		h.users.On("Create", mock.MatchedBy(func(p *user.CreateUserParams) bool {
			return p.Email == "new@example.com" && p.FirstName == "Ada" && p.LastName == "Lovelace"
		})).Return(created, nil)
		access := newAPIToken(1, "access")
		refresh := newAPIToken(2, "refresh")
		h.tokens.On("CreateToken", uint(55), apitoken.AccessToken).Return(access, nil)
		h.tokens.On("CreateToken", uint(55), apitoken.RefreshToken).Return(refresh, nil)

		got, err := h.svc.SignUp(&SignUpParams{Email: "new@example.com", FirstName: "Ada", LastName: "Lovelace"})

		require.NoError(t, err)
		assert.Same(t, access, got.AccessToken)
		assert.Same(t, refresh, got.RefreshToken)
		h.assertAllExpectations(t)
	})

	t.Run("user creation failure is wrapped, not passed through raw", func(t *testing.T) {
		h := newHarness()
		createErr := errors.New("email already exists")
		h.users.On("Create", mock.Anything).Return(nil, createErr)

		got, err := h.svc.SignUp(&SignUpParams{Email: "dup@example.com"})

		require.Nil(t, got)
		require.Error(t, err)
		assert.ErrorIs(t, err, createErr)
		assert.Contains(t, err.Error(), "failed to create user")
		h.tokens.AssertNotCalled(t, "CreateToken", mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("access token failure aborts before refresh token creation", func(t *testing.T) {
		h := newHarness()
		created := newUser(55)
		h.users.On("Create", mock.Anything).Return(created, nil)
		tokenErr := errors.New("token svc down")
		h.tokens.On("CreateToken", uint(55), apitoken.AccessToken).Return(nil, tokenErr)

		got, err := h.svc.SignUp(&SignUpParams{Email: "new@example.com"})

		require.Nil(t, got)
		require.ErrorIs(t, err, tokenErr)
		h.tokens.AssertNotCalled(t, "CreateToken", uint(55), apitoken.RefreshToken)
		h.assertAllExpectations(t)
	})

	t.Run("refresh token failure is returned", func(t *testing.T) {
		h := newHarness()
		created := newUser(55)
		h.users.On("Create", mock.Anything).Return(created, nil)
		h.tokens.On("CreateToken", uint(55), apitoken.AccessToken).Return(newAPIToken(1, "access"), nil)
		tokenErr := errors.New("refresh token svc down")
		h.tokens.On("CreateToken", uint(55), apitoken.RefreshToken).Return(nil, tokenErr)

		got, err := h.svc.SignUp(&SignUpParams{Email: "new@example.com"})

		require.Nil(t, got)
		require.ErrorIs(t, err, tokenErr)
		h.assertAllExpectations(t)
	})
}

func TestService_RefreshToken(t *testing.T) {
	t.Run("valid refresh token issues a new access/refresh pair", func(t *testing.T) {
		h := newHarness()
		uid := uint(9)
		h.tokens.On("ValidateToken", "valid-refresh", apitoken.RefreshToken).Return(uid, nil)
		access := newAPIToken(1, "access")
		newRefresh := newAPIToken(2, "refresh")
		h.tokens.On("CreateToken", uid, apitoken.AccessToken).Return(access, nil)
		h.tokens.On("CreateToken", uid, apitoken.RefreshToken).Return(newRefresh, nil)

		got, err := h.svc.RefreshToken("valid-refresh")

		require.NoError(t, err)
		assert.Same(t, access, got.AccessToken)
		assert.Same(t, newRefresh, got.RefreshToken)
		h.assertAllExpectations(t)
	})

	t.Run("invalid refresh token returns sentinel ErrInvalidRefreshToken, not the raw error", func(t *testing.T) {
		h := newHarness()
		rawErr := errors.New("token expired at row 12: signature mismatch")
		h.tokens.On("ValidateToken", "bad-token", apitoken.RefreshToken).Return(uint(0), rawErr)

		got, err := h.svc.RefreshToken("bad-token")

		require.Nil(t, got)
		require.ErrorIs(t, err, httpx.ErrInvalidRefreshToken)
		assert.NotErrorIs(t, err, rawErr, "raw validation error should not leak to the caller")
		h.tokens.AssertNotCalled(t, "CreateToken", mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("access token creation failure aborts before new refresh token is created", func(t *testing.T) {
		h := newHarness()
		uid := uint(9)
		h.tokens.On("ValidateToken", "valid-refresh", apitoken.RefreshToken).Return(uid, nil)
		tokenErr := errors.New("create failed")
		h.tokens.On("CreateToken", uid, apitoken.AccessToken).Return(nil, tokenErr)

		got, err := h.svc.RefreshToken("valid-refresh")

		require.Nil(t, got)
		require.ErrorIs(t, err, tokenErr)
		h.tokens.AssertNotCalled(t, "CreateToken", uid, apitoken.RefreshToken)
		h.assertAllExpectations(t)
	})

	t.Run("new refresh token creation failure is returned", func(t *testing.T) {
		h := newHarness()
		uid := uint(9)
		h.tokens.On("ValidateToken", "valid-refresh", apitoken.RefreshToken).Return(uid, nil)
		h.tokens.On("CreateToken", uid, apitoken.AccessToken).Return(newAPIToken(1, "access"), nil)
		tokenErr := errors.New("create failed")
		h.tokens.On("CreateToken", uid, apitoken.RefreshToken).Return(nil, tokenErr)

		got, err := h.svc.RefreshToken("valid-refresh")

		require.Nil(t, got)
		require.ErrorIs(t, err, tokenErr)
		h.assertAllExpectations(t)
	})
}

func TestService_Logout(t *testing.T) {
	t.Run("delegates directly to token manager Deactivate", func(t *testing.T) {
		h := newHarness()
		h.tokens.On("Deactivate", uint(3)).Return(nil)

		err := h.svc.Logout(3)

		require.NoError(t, err)
		h.assertAllExpectations(t)
	})

	t.Run("propagates deactivate error", func(t *testing.T) {
		h := newHarness()
		deactivateErr := errors.New("deactivate failed")
		h.tokens.On("Deactivate", uint(3)).Return(deactivateErr)

		err := h.svc.Logout(3)

		require.ErrorIs(t, err, deactivateErr)
		h.assertAllExpectations(t)
	})
}
