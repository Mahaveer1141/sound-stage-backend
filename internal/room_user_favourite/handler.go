package roomuserfavourite

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"sound-stage-backend/internal/pkg/current"
	"sound-stage-backend/internal/pkg/httpx"
)

type favouriteService interface {
	Add(userID, roomID uint) error
	Remove(userID, roomID uint) error
}

type Handler struct {
	service  favouriteService
	validate *validator.Validate
}

func NewHandler(service favouriteService) *Handler {
	return &Handler{service: service, validate: validator.New()}
}

type AddFavouriteInput struct {
	RoomID uint `json:"roomId" validate:"required"`
}

func (h *Handler) Add(c *gin.Context) {
	userID, _ := current.UserID(c)

	var input AddFavouriteInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if err := h.validate.Struct(input); err != nil {
		httpx.ErrorResponse(c, http.StatusUnprocessableEntity, "Validation error: "+err.Error())
		return
	}

	if err := h.service.Add(userID, input.RoomID); err != nil {
		if errors.Is(err, httpx.ErrUserBlocked) {
			httpx.ErrorResponse(c, http.StatusForbidden, "You are blocked from this room")
			return
		}
		httpx.ErrorResponse(c, http.StatusUnprocessableEntity, "Failed to add favorite")
		return
	}

	httpx.SuccessResponse(c, http.StatusOK, "Room added to favorites", nil)
}

func (h *Handler) Remove(c *gin.Context) {
	userID, _ := current.UserID(c)

	roomIDStr := c.Param("roomId")
	roomID, err := strconv.Atoi(roomIDStr)
	if err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid room ID")
		return
	}

	if err := h.service.Remove(userID, uint(roomID)); err != nil {
		httpx.ErrorResponse(c, http.StatusUnprocessableEntity, "Failed to remove favorite")
		return
	}

	httpx.SuccessResponse(c, http.StatusOK, "Room removed from favorites", nil)
}
