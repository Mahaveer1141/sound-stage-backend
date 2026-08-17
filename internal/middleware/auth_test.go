package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	apitoken "sound-stage-backend/internal/api_token"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockTokenValidator struct {
	mock.Mock
}

func (m *MockTokenValidator) ValidateToken(token string, tokenType apitoken.TokenType) (uint, error) {
	args := m.Called(token, tokenType)
	return args.Get(0).(uint), args.Error(1)
}

func setupRouter(validator TokenValidator, nextCalled *bool, capturedExists *bool, capturedUserID *any) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AuthMiddleware(validator))
	r.GET("/protected", func(c *gin.Context) {
		if nextCalled != nil {
			*nextCalled = true
		}
		*capturedUserID, *capturedExists = c.Get("userId")
		c.JSON(http.StatusOK, gin.H{"userId": *capturedUserID})
	})
	return r
}

func TestAuthMiddleware_TableDriven(t *testing.T) {
	tests := []struct {
		name                 string
		authHeader           string
		queryToken           string
		mockCalled           bool
		mockToken            string
		mockReturn           uint
		mockErr              error
		expectedStatus       int
		expectNextCalled     bool
		expectedBodyContains string
	}{
		{
			name:                 "success: valid bearer token",
			authHeader:           "Bearer valid-token-123",
			mockCalled:           true,
			mockToken:            "valid-token-123",
			mockReturn:           42,
			expectedStatus:       http.StatusOK,
			expectNextCalled:     true,
			expectedBodyContains: `"userId":42`,
		},
		{
			name:             "success: token via query param",
			queryToken:       "query-token-456",
			mockCalled:       true,
			mockToken:        "query-token-456",
			mockReturn:       7,
			expectedStatus:   http.StatusOK,
			expectNextCalled: true,
		},
		{
			name:             "success: bearer header takes precedence over query token",
			authHeader:       "Bearer header-token",
			queryToken:       "query-token",
			mockCalled:       true,
			mockToken:        "header-token",
			mockReturn:       1,
			expectedStatus:   http.StatusOK,
			expectNextCalled: true,
		},
		{
			name:             "failure: no token anywhere",
			mockCalled:       false,
			expectedStatus:   http.StatusUnauthorized,
			expectNextCalled: false,
		},
		{
			name:                 "failure: validator rejects token",
			authHeader:           "Bearer bad-token",
			mockCalled:           true,
			mockToken:            "bad-token",
			mockReturn:           0,
			mockErr:              errors.New("invalid token"),
			expectedStatus:       http.StatusUnauthorized,
			expectNextCalled:     false,
			expectedBodyContains: "invalid token",
		},
		{
			name:             "failure: empty bearer and no query token",
			authHeader:       "Bearer ",
			mockCalled:       false,
			expectedStatus:   http.StatusUnauthorized,
			expectNextCalled: false,
		},
		{
			name:             "edge case: empty bearer falls back to query token",
			authHeader:       "Bearer ",
			queryToken:       "fallback-query-token",
			mockCalled:       true,
			mockToken:        "fallback-query-token",
			mockReturn:       9,
			expectedStatus:   http.StatusOK,
			expectNextCalled: true,
		},
		{
			name:                 "edge case: header without Bearer prefix is passed through raw",
			authHeader:           "raw-token-no-prefix",
			mockCalled:           true,
			mockToken:            "raw-token-no-prefix",
			mockReturn:           0,
			mockErr:              errors.New("invalid token"),
			expectedStatus:       http.StatusUnauthorized,
			expectNextCalled:     false,
			expectedBodyContains: "invalid token",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockValidator := new(MockTokenValidator)
			if tc.mockCalled {
				mockValidator.On("ValidateToken", tc.mockToken, apitoken.AccessToken).
					Return(tc.mockReturn, tc.mockErr)
			}

			nextCalled := false
			var capturedUserID any
			var capturedExists bool
			router := setupRouter(mockValidator, &nextCalled, &capturedExists, &capturedUserID)

			target := "/protected"
			if tc.queryToken != "" {
				target += "?token=" + tc.queryToken
			}
			req := httptest.NewRequest(http.MethodGet, target, nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
			assert.Equal(t, tc.expectNextCalled, nextCalled)
			if tc.expectedBodyContains != "" {
				assert.Contains(t, w.Body.String(), tc.expectedBodyContains)
			}

			expectedUserIDSet := tc.mockCalled && tc.mockErr == nil
			assert.Equal(t, expectedUserIDSet, capturedExists)
			if expectedUserIDSet {
				assert.Equal(t, tc.mockReturn, capturedUserID)
			}

			if tc.mockCalled {
				mockValidator.AssertExpectations(t)
			} else {
				mockValidator.AssertNotCalled(t, "ValidateToken", mock.Anything, mock.Anything)
			}
		})
	}
}
