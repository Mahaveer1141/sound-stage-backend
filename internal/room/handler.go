package room

import (
	"context"
	"errors"
	"net/http"
	"sound-stage-backend/internal/pkg/current"
	"sound-stage-backend/internal/pkg/httpx"
	"sound-stage-backend/internal/pkg/listopts"
	"sound-stage-backend/internal/ws"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

type roomService interface {
	List(filter RoomFilter, sort listopts.Sort, p listopts.Pagination) ([]Room, int64, error)
	FindByID(id, userID uint) (*Room, error)
	Create(input *CreateRoomParams) (*Room, error)
	Update(id, userID uint, input *UpdateRoomParams) (*Room, error)
	Delete(ctx context.Context, id, userID uint) error
	ViewerContext(roomID, userID uint) (*RoomViewer, error)
	ViewerContexts(roomIDs []uint, userID uint) (map[uint]*RoomViewer, error)
	UpdatePrivateCode(roomID, userID uint) error
}

type webSocketBroadcaster interface {
	BroadcastToRoom(roomID uint, eventName ws.EventName, payload any)
}

type Handler struct {
	service  roomService
	validate *validator.Validate
	hub      webSocketBroadcaster
}

func NewHandler(service roomService, hub webSocketBroadcaster) *Handler {
	return &Handler{service: service, validate: httpx.NewValidator(), hub: hub}
}

func (h *Handler) Create(c *gin.Context) {
	userId, _ := current.UserID(c)
	var input CreateRoomParams
	if err := bindRoomRequest(c, &input); err != nil {
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

	viewer, _ := h.service.ViewerContext(room.ID, userId)
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
	if err := bindRoomRequest(c, &input); err != nil {
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

	h.hub.BroadcastToRoom(room.ID, ws.EventChatEnabledUpdated, gin.H{"isChatEnabled": room.IsChatEnabled})

	viewer, _ := h.service.ViewerContext(room.ID, userId)
	httpx.SuccessResponse(c, http.StatusOK, "Room updated successfully", BuildRoomResponse(room, viewer))
}

func (h *Handler) List(c *gin.Context) {
	var p listopts.Pagination
	if err := c.ShouldBindQuery(&p); err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid pagination params")
		return
	}
	if p.Page <= 0 || p.PageSize <= 0 || p.PageSize > 50 {
		httpx.ErrorResponse(c, http.StatusBadRequest, "page and pageSize must be positive and pageSize must be less than 50")
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

	roomIDs := make([]uint, len(rooms))
	for i := range rooms {
		roomIDs[i] = rooms[i].ID
	}
	viewers, _ := h.service.ViewerContexts(roomIDs, filter.UserID)

	responses := BuildRoomListResponse(rooms, viewers)

	httpx.PaginatedSuccessResponse(c, "Rooms fetched successfully", responses, p.Page, p.PageSize, int(count))
}

func (h *Handler) FindByID(c *gin.Context) {
	id := c.Param("id")
	roomId, err := strconv.Atoi(id)
	if err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid room ID")
		return
	}

	currentUserID, _ := current.UserID(c)
	room, err := h.service.FindByID(uint(roomId), currentUserID)
	if err != nil {
		if errors.Is(err, httpx.ErrForbidden) {
			httpx.ErrorResponse(c, http.StatusForbidden, "You are not allowed to view this room")
			return
		}
		httpx.ErrorResponse(c, http.StatusNotFound, "Room not found")
		return
	}

	viewer, _ := h.service.ViewerContext(room.ID, currentUserID)

	httpx.SuccessResponse(c, http.StatusOK, "Room fetched successfully", BuildRoomResponse(room, viewer))
}

func (h *Handler) Delete(c *gin.Context) {
	id := c.Param("id")
	roomId, err := strconv.Atoi(id)
	if err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid room ID")
		return
	}
	userId, _ := current.UserID(c)

	if err := h.service.Delete(c.Request.Context(), uint(roomId), userId); err != nil {
		if errors.Is(err, httpx.ErrForbidden) {
			httpx.ErrorResponse(c, http.StatusForbidden, "Only the room owner can delete this room")
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, httpx.ErrRecordNotFound) {
			httpx.ErrorResponse(c, http.StatusNotFound, "Room not found")
			return
		}
		httpx.ErrorResponse(c, http.StatusUnprocessableEntity, "Failed to delete room")
		return
	}

	h.hub.BroadcastToRoom(uint(roomId), ws.EventRoomDeleted, gin.H{"roomId": roomId})

	httpx.SuccessResponse(c, http.StatusOK, "Room deleted successfully", nil)
}

func (h *Handler) UpdatePrivateCode(c *gin.Context) {
	id := c.Param("id")
	roomId, err := strconv.Atoi(id)
	if err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid room ID")
		return
	}
	currentUserID, _ := current.UserID(c)

	err = h.service.UpdatePrivateCode(uint(roomId), currentUserID)
	if err != nil {
		if errors.Is(err, httpx.ErrForbidden) {
			httpx.ErrorResponse(c, http.StatusForbidden, "You are not allowed to update this room's private code")
			return
		}
		httpx.ErrorResponse(c, http.StatusUnprocessableEntity, "Failed to update private code")
		return
	}

	httpx.SuccessResponse(c, http.StatusOK, "Private code updated successfully", nil)
}

func bindRoomRequest(c *gin.Context, obj any) error {
	if strings.HasPrefix(c.ContentType(), binding.MIMEMultipartPOSTForm) {
		return c.ShouldBindWith(obj, binding.FormMultipart)
	}
	return c.ShouldBind(obj)
}
