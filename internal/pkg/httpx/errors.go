package httpx

import "errors"

// Auth errors
var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden: user doesn't have permission")
)

// API Token errors
var (
	ErrInvalidOrExpiredToken  = errors.New("invalid or expired token")
	ErrInvalidTokenClaims     = errors.New("invalid token claims")
	ErrTokenNotFoundOrRevoked = errors.New("token not found or revoked")
	ErrTokenDataMismatch      = errors.New("token data mismatch")
	ErrInvalidRefreshToken    = errors.New("Invalid Refresh Token")
)

// OTP Request errors
var (
	ErrInvalidOTP            = errors.New("Invalid OTP")
	ErrUserOrEmailRequired   = errors.New("either user_id or email must be provided")
	ErrOTPRequestAlreadyMade = errors.New("OTP request already made within the last minute, please wait")
	ErrOTPNotVerified        = errors.New("OTP not verified for this email")
)

// Database/Record errors
var (
	ErrRecordNotFound = errors.New("record not found")
)

// Room errors
var (
	ErrUserBlocked = errors.New("user is blocked from this room")
)

// Chat message errors
var (
	ErrPinnedLimitReached = errors.New("pinned messages limit reached")
)

// WebRTC errors
var (
	ErrPeerConnectionNotFound = errors.New("peer connection not found")
)
