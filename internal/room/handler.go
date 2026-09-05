package room

import (
	"errors"
	"net/http"
	"sound-stage-backend/internal/pkg/current"
	"sound-stage-backend/internal/pkg/httpx"
	"sound-stage-backend/internal/pkg/listopts"
	roomuser "sound-stage-backend/internal/room_user"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type roomService interface {
	List(filter RoomFilter, sort listopts.Sort, p listopts.Pagination) ([]Room, int64, error)
	FindByID(id uint) (*Room, error)
	Create(input *CreateRoomParams) (*Room, error)
	Update(id, userID uint, input *UpdateRoomParams) (*Room, error)
	CurrentRoomUser(roomID uint, userID uint) (*roomuser.RoomUser, error)
	UpdatePrivateCode(roomID uint) error
}

type Handler struct {
	service  roomService
	validate *validator.Validate
}

func NewHandler(service roomService) *Handler {
	return &Handler{service: service, validate: validator.New()}
}

func (h *Handler) Create(c *gin.Context) {
	userId, _ := current.UserID(c)
	var input CreateRoomParams
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid request payload")
		return
	}
	input.CreatorID = userId
	if err := h.validate.Struct(input); err != nil {
		httpx.ErrorResponse(c, http.StatusUnprocessableEntity, "Validation error: "+err.Error())
		return
	}

	room, err := h.service.Create(&input)
	if err != nil {
		httpx.ErrorResponse(c, http.StatusUnprocessableEntity, "Failed to create room")
		return
	}

	viewer, _ := h.service.CurrentRoomUser(room.ID, userId)
	httpx.SuccessResponse(c, http.StatusOK, "Room created successfully", BuildRoomResponse(room, viewer))
}

func (h *Handler) Update(c *gin.Context) {
	id := c.Param("id")
	roomId, err := strconv.Atoi(id)
	if err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid room ID")
		return
	}
	userId, _ := current.UserID(c)

	var input UpdateRoomParams
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := h.validate.Struct(input); err != nil {
		httpx.ErrorResponse(c, http.StatusUnprocessableEntity, "Validation error: "+err.Error())
		return
	}

	room, err := h.service.Update(uint(roomId), userId, &input)
	if err != nil {
		if errors.Is(err, httpx.ErrForbidden) {
			httpx.ErrorResponse(c, http.StatusForbidden, "You are not allowed to update this room")
			return
		}
		httpx.ErrorResponse(c, http.StatusUnprocessableEntity, "Failed to update room")
		return
	}

	viewer, _ := h.service.CurrentRoomUser(room.ID, userId)
	httpx.SuccessResponse(c, http.StatusOK, "Room updated successfully", BuildRoomResponse(room, viewer))
}

func (h *Handler) List(c *gin.Context) {
	var p listopts.Pagination
	if err := c.ShouldBindQuery(&p); err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid pagination params")
		return
	}
	if p.Page <= 0 || p.PageSize <= 0 {
		httpx.ErrorResponse(c, http.StatusBadRequest, "page and pageSize must be positive")
		return
	}

	var filter RoomFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid filter")
		return
	}
	filter.UserID, _ = current.UserID(c)

	var sort listopts.Sort
	if err := c.ShouldBindQuery(&sort); err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid sort")
		return
	}

	rooms, count, err := h.service.List(filter, sort, p)
	if err != nil {
		httpx.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch rooms")
		return
	}

	responses := BuildRoomListResponse(rooms, nil)

	httpx.PaginatedSuccessResponse(c, "Rooms fetched successfully", responses, p.Page, p.PageSize, int(count))
}

func (h *Handler) FindByID(c *gin.Context) {
	id := c.Param("id")
	roomId, err := strconv.Atoi(id)
	if err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid room ID")
		return
	}

	room, err := h.service.FindByID(uint(roomId))
	if err != nil {
		httpx.ErrorResponse(c, http.StatusNotFound, "Failed to fetch room")
		return
	}

	currentUserID, _ := current.UserID(c)
	viewerRoomUser, _ := h.service.CurrentRoomUser(room.ID, currentUserID)

	httpx.SuccessResponse(c, http.StatusOK, "Room fetched successfully", BuildRoomResponse(room, viewerRoomUser))
}

func (h *Handler) UpdatePrivateCode(c *gin.Context) {
	id := c.Param("id")
	roomId, err := strconv.Atoi(id)
	if err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid room ID")
		return
	}

	err = h.service.UpdatePrivateCode(uint(roomId))
	if err != nil {
		httpx.ErrorResponse(c, http.StatusUnprocessableEntity, "Failed to update private code")
		return
	}

	httpx.SuccessResponse(c, http.StatusOK, "Private code updated successfully", nil)
}
