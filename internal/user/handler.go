package user

import (
	"net/http"
	"sound-stage-backend/internal/pkg/httpx"

	"github.com/gin-gonic/gin"
)

type Handler interface {
	CurrentUser(c *gin.Context)
	UpdateProfile(c *gin.Context)
}

type handler struct {
	service Service
}

func NewHandler(service Service) Handler {
	return &handler{service: service}
}

func (h *handler) CurrentUser(c *gin.Context) {
	userId, _ := c.Get("userId")
	user, err := h.service.FindByID(userId.(uint))
	if err != nil {
		httpx.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch user")
		return
	}

	httpx.SuccessResponse(c, http.StatusOK, "User fetched successfully", user)
}

func (h *handler) UpdateProfile(c *gin.Context) {
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
