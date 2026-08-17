package auth

import (
	"errors"
	"net/http"
	apitoken "sound-stage-backend/internal/api_token"
	otprequest "sound-stage-backend/internal/otp_request"
	"sound-stage-backend/internal/pkg/testutil"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockAuthService struct {
	mock.Mock
}

func (m *mockAuthService) RequestOTP(email string) (*otprequest.OTPRequest, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*otprequest.OTPRequest), args.Error(1)
}

func (m *mockAuthService) VerifyOTP(params VerifyOTPParams) (*apitoken.TokenResult, error) {
	args := m.Called(params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*apitoken.TokenResult), args.Error(1)
}

func (m *mockAuthService) SignUp(input *SignUpParams) (*apitoken.TokenResult, error) {
	args := m.Called(input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*apitoken.TokenResult), args.Error(1)
}

func (m *mockAuthService) RefreshToken(refreshToken string) (*apitoken.TokenResult, error) {
	args := m.Called(refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*apitoken.TokenResult), args.Error(1)
}

func (m *mockAuthService) Logout(userID uint) error {
	args := m.Called(userID)
	return args.Error(0)
}

func init() {
	gin.SetMode(gin.TestMode)
}

func TestHandler_RequestOTP(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		mockSetup      func(m *mockAuthService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "success lowercases email",
			body: `{"email":"User@Example.com"}`,
			mockSetup: func(m *mockAuthService) {
				m.On("RequestOTP", "user@example.com").
					Return(&otprequest.OTPRequest{}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "OTP requested successfully",
		},
		{
			name:           "invalid json body",
			body:           `{invalid`,
			mockSetup:      func(m *mockAuthService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request payload",
		},
		{
			name:           "missing email fails validation",
			body:           `{}`,
			mockSetup:      func(m *mockAuthService) {},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   "Validation error",
		},
		{
			name: "service error is propagated",
			body: `{"email":"user@example.com"}`,
			mockSetup: func(m *mockAuthService) {
				m.On("RequestOTP", "user@example.com").
					Return(nil, errors.New("rate limited"))
			},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   "Failed to request OTP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := new(mockAuthService)
			tt.mockSetup(mockSvc)
			h := NewHandler(mockSvc)

			w, c := testutil.NewTestContext(http.MethodPost, "/otp/request", tt.body)
			h.RequestOTP(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)
			mockSvc.AssertExpectations(t)
		})
	}
}

func TestHandler_VerifyOTP(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		mockSetup      func(m *mockAuthService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "success returns tokens",
			body: `{"email":"User@Example.com","otp":"123456"}`,
			mockSetup: func(m *mockAuthService) {
				m.On("VerifyOTP", VerifyOTPParams{Email: "user@example.com", OTP: "123456"}).
					Return(&apitoken.TokenResult{}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "OTP verified successfully",
		},
		{
			name:           "invalid json body",
			body:           `{invalid`,
			mockSetup:      func(m *mockAuthService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request payload",
		},
		{
			name:           "missing otp fails validation",
			body:           `{"email":"user@example.com"}`,
			mockSetup:      func(m *mockAuthService) {},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   "Failed to verify OTP",
		},
		{
			name: "service error is propagated",
			body: `{"email":"user@example.com","otp":"000000"}`,
			mockSetup: func(m *mockAuthService) {
				m.On("VerifyOTP", VerifyOTPParams{Email: "user@example.com", OTP: "000000"}).
					Return(nil, errors.New("otp expired"))
			},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   "Failed to verify OTP",
		},
		{
			name: "nil result returns 200 with empty tokens",
			body: `{"email":"user@example.com","otp":"123456"}`,
			mockSetup: func(m *mockAuthService) {
				m.On("VerifyOTP", VerifyOTPParams{Email: "user@example.com", OTP: "123456"}).
					Return(nil, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "OTP verified successfully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := new(mockAuthService)
			tt.mockSetup(mockSvc)
			h := NewHandler(mockSvc)

			w, c := testutil.NewTestContext(http.MethodPost, "/otp/verify", tt.body)
			h.VerifyOTP(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)
			mockSvc.AssertExpectations(t)
		})
	}
}

func TestHandler_RefreshToken(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		mockSetup      func(m *mockAuthService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "success",
			body: `{"refreshToken":"valid-token"}`,
			mockSetup: func(m *mockAuthService) {
				m.On("RefreshToken", "valid-token").
					Return(&apitoken.TokenResult{}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "Token refreshed successfully",
		},
		{
			name:           "invalid json body",
			body:           `{invalid`,
			mockSetup:      func(m *mockAuthService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request payload",
		},
		{
			name:           "missing refreshToken fails validation",
			body:           `{}`,
			mockSetup:      func(m *mockAuthService) {},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   "Failed to refresh token",
		},
		{
			name: "service error is propagated",
			body: `{"refreshToken":"expired-token"}`,
			mockSetup: func(m *mockAuthService) {
				m.On("RefreshToken", "expired-token").
					Return(nil, errors.New("token expired"))
			},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   "Failed to refresh token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := new(mockAuthService)
			tt.mockSetup(mockSvc)
			h := NewHandler(mockSvc)

			w, c := testutil.NewTestContext(http.MethodPost, "/auth/refresh", tt.body)
			h.RefreshToken(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)
			mockSvc.AssertExpectations(t)
		})
	}
}

func TestHandler_SignUp(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		mockSetup      func(m *mockAuthService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "success",
			body: `{"email":"user@example.com","firstName":"Jane","lastName":"Doe"}`,
			mockSetup: func(m *mockAuthService) {
				m.On("SignUp", mock.MatchedBy(func(input *SignUpParams) bool {
					return input.Email == "user@example.com" && input.FirstName == "Jane"
				})).Return(&apitoken.TokenResult{}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "User Signed Up successfully",
		},
		{
			name:           "invalid json body",
			body:           `{invalid`,
			mockSetup:      func(m *mockAuthService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request body",
		},
		{
			name:           "missing required firstName fails validation",
			body:           `{"email":"user@example.com"}`,
			mockSetup:      func(m *mockAuthService) {},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   "Validation error",
		},
		{
			name: "service error is propagated (e.g. duplicate email)",
			body: `{"email":"user@example.com","firstName":"Jane"}`,
			mockSetup: func(m *mockAuthService) {
				m.On("SignUp", mock.AnythingOfType("*auth.SignUpParams")).
					Return(nil, errors.New("email already registered"))
			},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   "Failed to create user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := new(mockAuthService)
			tt.mockSetup(mockSvc)
			h := NewHandler(mockSvc)

			w, c := testutil.NewTestContext(http.MethodPost, "/auth/signup", tt.body)
			h.SignUp(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)
			mockSvc.AssertExpectations(t)
		})
	}
}

func TestHandler_Logout(t *testing.T) {
	tests := []struct {
		name           string
		userID         uint
		mockSetup      func(m *mockAuthService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:   "success",
			userID: 42,
			mockSetup: func(m *mockAuthService) {
				m.On("Logout", uint(42)).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "Logout successful",
		},
		{
			name:   "service error is propagated",
			userID: 42,
			mockSetup: func(m *mockAuthService) {
				m.On("Logout", uint(42)).Return(errors.New("db error"))
			},
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   "Failed to logout",
		},
		{
			name:   "no userId in context defaults to zero value",
			userID: 0,
			mockSetup: func(m *mockAuthService) {
				m.On("Logout", uint(0)).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "Logout successful",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := new(mockAuthService)
			tt.mockSetup(mockSvc)
			h := NewHandler(mockSvc)

			w, c := testutil.NewTestContext(http.MethodPost, "/auth/logout", "")
			if tt.name != "no userId in context defaults to zero value" {
				c.Set("userId", tt.userID)
			}

			h.Logout(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)
			mockSvc.AssertExpectations(t)
		})
	}
}
