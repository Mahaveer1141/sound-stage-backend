package user

import (
	"net/http"
	"sound-stage-backend/internal/pkg/current"
	"sound-stage-backend/internal/pkg/httpx"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

type userService interface {
	FindByID(userId uint) (*User, error)
	UpdateProfile(id uint, input *UpdateUserParams) (*User, error)
}

type Handler struct {
	service  userService
	validate *validator.Validate
}

func NewHandler(service userService) *Handler {
	return &Handler{service: service, validate: httpx.NewValidator()}
}

func (h *Handler) CurrentUser(c *gin.Context) {
	userID, _ := current.UserID(c)
	u, err := h.service.FindByID(userID)
	if err != nil {
		httpx.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch user")
		return
	}

	httpx.SuccessResponse(c, http.StatusOK, "User fetched successfully", BuildUserResponse(u, true))
}

func (h *Handler) UpdateProfile(c *gin.Context) {
	userId, _ := current.UserID(c)

	var input UpdateUserParams
	if err := bindRequest(c, &input); err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if err := h.validate.Struct(input); err != nil {
		httpx.ErrorResponse(c, http.StatusUnprocessableEntity, "Validation error: "+err.Error())
		return
	}

	u, err := h.service.UpdateProfile(userId, &input)
	if err != nil {
		httpx.ErrorResponse(c, http.StatusInternalServerError, "Failed to update profile")
		return
	}

	httpx.SuccessResponse(c, http.StatusOK, "Profile updated successfully", BuildUserResponse(u, true))
}

func bindRequest(c *gin.Context, obj any) error {
	if strings.HasPrefix(c.ContentType(), binding.MIMEMultipartPOSTForm) {
		return c.ShouldBindWith(obj, binding.FormMultipart)
	}
	return c.ShouldBind(obj)
}
