package httpx

import "errors"

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
)

// Database/Record errors
var (
	ErrRecordNotFound = errors.New("record not found")
)

// WebRTC errors
var (
	ErrPeerConnectionNotFound = errors.New("peer connection not found")
)
