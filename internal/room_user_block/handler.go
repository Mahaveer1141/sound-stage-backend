package roomuserblock

import (
	"errors"
	"net/http"
	"sound-stage-backend/internal/pkg/current"
	"sound-stage-backend/internal/pkg/httpx"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type blockService interface {
	Add(roomID, userID, blockedByID uint) error
	Remove(roomID, userID, blockedByID uint) error
}

type Handler struct {
	service  blockService
	validate *validator.Validate
}

func NewHandler(service blockService) *Handler {
	return &Handler{service: service, validate: validator.New()}
}

type AddBlockInput struct {
	UserID uint `json:"userId" validate:"required"`
}

func (h *Handler) Add(c *gin.Context) {
	blockedByID, _ := current.UserID(c)

	roomIDStr := c.Param("id")
	roomID, err := strconv.Atoi(roomIDStr)
	if err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid room ID")
		return
	}

	var input AddBlockInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := h.validate.Struct(input); err != nil {
		httpx.ErrorResponse(c, http.StatusUnprocessableEntity, "Validation error: "+err.Error())
		return
	}

	if err := h.service.Add(uint(roomID), input.UserID, blockedByID); err != nil {
		if errors.Is(err, httpx.ErrForbidden) {
			httpx.ErrorResponse(c, http.StatusForbidden, "You are not allowed to block users in this room")
			return
		}
		httpx.ErrorResponse(c, http.StatusUnprocessableEntity, "Failed to block user")
		return
	}

	httpx.SuccessResponse(c, http.StatusOK, "User blocked successfully", nil)
}

func (h *Handler) Remove(c *gin.Context) {
	blockedByID, _ := current.UserID(c)

	roomIDStr := c.Param("id")
	roomID, err := strconv.Atoi(roomIDStr)
	if err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid room ID")
		return
	}

	userIDStr := c.Param("userId")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	if err := h.service.Remove(uint(roomID), uint(userID), blockedByID); err != nil {
		if errors.Is(err, httpx.ErrForbidden) {
			httpx.ErrorResponse(c, http.StatusForbidden, "You are not allowed to unblock users in this room")
			return
		}
		httpx.ErrorResponse(c, http.StatusUnprocessableEntity, "Failed to unblock user")
		return
	}

	httpx.SuccessResponse(c, http.StatusOK, "User unblocked successfully", nil)
}
