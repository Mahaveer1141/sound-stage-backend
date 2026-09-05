package roomuserfavourite

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"sound-stage-backend/internal/pkg/current"
	"sound-stage-backend/internal/pkg/httpx"
	"sound-stage-backend/internal/pkg/listopts"
	"sound-stage-backend/internal/room"
)

type favouriteService interface {
	Add(userID, roomID uint) error
	Remove(userID, roomID uint) error
	ListUserFavourites(userID uint, p listopts.Pagination) ([]RoomUserFavourite, int64, error)
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

func (h *Handler) ListUserFavourites(c *gin.Context) {
	var p listopts.Pagination
	if err := c.ShouldBindQuery(&p); err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid pagination params")
		return
	}
	if p.Page <= 0 || p.PageSize <= 0 {
		httpx.ErrorResponse(c, http.StatusBadRequest, "page and pageSize must be positive")
		return
	}

	userID, _ := current.UserID(c)
	favourites, count, err := h.service.ListUserFavourites(userID, p)
	if err != nil {
		httpx.ErrorResponse(c, http.StatusUnprocessableEntity, "Failed to list favorites")
		return
	}

	response := make([]room.RoomResponse, len(favourites))
	for i := range favourites {
		response[i] = room.BuildRoomResponse(&favourites[i].Room, nil)
	}

	httpx.PaginatedSuccessResponse(c, "User favourites fetched successfully", response, p.Page, p.PageSize, int(count))
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
