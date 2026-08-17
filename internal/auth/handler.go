package auth

import (
	"net/http"
	apitoken "sound-stage-backend/internal/api_token"
	otprequest "sound-stage-backend/internal/otp_request"
	"sound-stage-backend/internal/pkg/httpx"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type authService interface {
	RequestOTP(email string) (*otprequest.OTPRequest, error)
	VerifyOTP(params VerifyOTPParams) (*apitoken.TokenResult, error)
	SignUp(input *SignUpParams) (*apitoken.TokenResult, error)
	RefreshToken(refreshToken string) (*apitoken.TokenResult, error)
	Logout(userID uint) error
}

type Handler struct {
	service  authService
	validate *validator.Validate
}

type RequestOTPParams struct {
	Email string `json:"email" validate:"required,email"`
}

type VerifyOTPParams struct {
	Email string `json:"email" validate:"required,email"`
	OTP   string `json:"otp" validate:"required"`
}

type RefreshTokenParams struct {
	RefreshToken string `json:"refreshToken" validate:"required"`
}

type SignUpParams struct {
	Email     string `json:"email" validate:"required,email"`
	FirstName string `json:"firstName" validate:"required"`
	LastName  string `json:"lastName,omitempty"`
}

func NewHandler(service authService) *Handler {
	return &Handler{service: service, validate: validator.New()}
}

func (h *Handler) RequestOTP(c *gin.Context) {
	var input RequestOTPParams

	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := h.validate.Struct(input); err != nil {
		httpx.ErrorResponse(c, http.StatusUnprocessableEntity, "Validation error: "+err.Error())
		return
	}

	email := strings.ToLower(input.Email)
	_, err := h.service.RequestOTP(email)

	if err != nil {
		httpx.ErrorResponse(c, http.StatusUnprocessableEntity, "Failed to request OTP: "+err.Error())
		return
	}

	httpx.SuccessResponse(c, http.StatusOK, "OTP requested successfully", nil)
}

func (h *Handler) VerifyOTP(c *gin.Context) {
	var input VerifyOTPParams

	if err := c.ShouldBindBodyWithJSON(&input); err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := h.validate.Struct(input); err != nil {
		httpx.ErrorResponse(c, http.StatusUnprocessableEntity, "Failed to verify OTP: "+err.Error())
		return
	}

	input.Email = strings.ToLower(input.Email)
	result, err := h.service.VerifyOTP(input)
	if err != nil {
		httpx.ErrorResponse(c, http.StatusUnprocessableEntity, "Failed to verify OTP: "+err.Error())
		return
	}

	if result == nil {
		httpx.SuccessResponse(c, http.StatusOK, "OTP verified successfully", apitoken.TokenResponse{
			AccessToken:  "",
			RefreshToken: "",
		})
		return
	}

	httpx.SuccessResponse(c, http.StatusOK, "OTP verified successfully", apitoken.ToTokenResponse(result))
}

func (h *Handler) RefreshToken(c *gin.Context) {
	var input RefreshTokenParams

	if err := c.ShouldBindBodyWithJSON(&input); err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := h.validate.Struct(input); err != nil {
		httpx.ErrorResponse(c, http.StatusUnprocessableEntity, "Failed to refresh token: "+err.Error())
		return
	}

	result, err := h.service.RefreshToken(input.RefreshToken)
	if err != nil {
		httpx.ErrorResponse(c, http.StatusUnprocessableEntity, "Failed to refresh token: "+err.Error())
		return
	}

	httpx.SuccessResponse(c, http.StatusOK, "Token refreshed successfully", apitoken.ToTokenResponse(result))
}

func (h *Handler) Logout(c *gin.Context) {
	userId := c.GetUint("userId")
	err := h.service.Logout(userId)
	if err != nil {
		httpx.ErrorResponse(c, http.StatusUnprocessableEntity, "Failed to logout: "+err.Error())
		return
	}

	httpx.SuccessResponse(c, http.StatusOK, "Logout successful", nil)
}

func (h *Handler) SignUp(c *gin.Context) {
	var input SignUpParams

	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.validate.Struct(input); err != nil {
		httpx.ErrorResponse(c, http.StatusUnprocessableEntity, "Validation error: "+err.Error())
		return
	}

	result, err := h.service.SignUp(&input)
	if err != nil {
		httpx.ErrorResponse(c, http.StatusUnprocessableEntity, "Failed to create user: "+err.Error())
		return
	}

	httpx.SuccessResponse(c, http.StatusOK, "User Signed Up successfully", apitoken.ToTokenResponse(result))
}
