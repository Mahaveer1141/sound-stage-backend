package user

import (
	"net/http"
	"sound-stage-backend/internal/pkg/httpx"

	"github.com/gin-gonic/gin"
)

type userService interface {
	FindByID(userId uint) (*User, error)
	UpdateProfile(id uint, input *UpdateUserParams) (*User, error)
}

type Handler struct {
	service userService
}

func NewHandler(service userService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CurrentUser(c *gin.Context) {
	userId, _ := c.Get("userId")
	user, err := h.service.FindByID(userId.(uint))
	if err != nil {
		httpx.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch user")
		return
	}

	httpx.SuccessResponse(c, http.StatusOK, "User fetched successfully", user)
}

func (h *Handler) UpdateProfile(c *gin.Context) {
	userId, _ := c.Get("userId")

	var input UpdateUserParams
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid request payload")
		return
	}

	user, err := h.service.UpdateProfile(userId.(uint), &input)
	if err != nil {
		httpx.ErrorResponse(c, http.StatusInternalServerError, "Failed to update profile")
		return
	}

	httpx.SuccessResponse(c, http.StatusOK, "Profile updated successfully", user)
}
