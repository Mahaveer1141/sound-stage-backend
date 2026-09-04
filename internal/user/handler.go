package user

import (
	"net/http"
	"sound-stage-backend/internal/pkg/current"
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
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid request payload")
		return
	}

	u, err := h.service.UpdateProfile(userId, &input)
	if err != nil {
		httpx.ErrorResponse(c, http.StatusInternalServerError, "Failed to update profile")
		return
	}

	httpx.SuccessResponse(c, http.StatusOK, "Profile updated successfully", BuildUserResponse(u, true))
}
