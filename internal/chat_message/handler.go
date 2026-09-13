package chatmessage

import (
	"errors"
	"net/http"
	"strconv"

	"sound-stage-backend/internal/pkg/current"
	"sound-stage-backend/internal/pkg/httpx"
	"sound-stage-backend/internal/pkg/listopts"
	"sound-stage-backend/internal/ws"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type webSocketHub interface {
	BroadcastToRoom(roomID uint, eventName ws.EventName, payload any)
}

type chatMessageService interface {
	Create(input *CreateChatMessageParams) (*ChatMessage, error)
	List(userID uint, filter ChatMessageFilter, p listopts.Pagination) ([]ChatMessage, int64, error)
	SetPinned(userID, roomID, messageID uint, pinned bool) (*ChatMessage, error)
}

type createMessageRequest struct {
	Content  string `json:"content" validate:"required"`
	IsPinned bool   `json:"isPinned"`
}

type setPinnedRequest struct {
	IsPinned bool `json:"isPinned"`
}

type Handler struct {
	service  chatMessageService
	hub      webSocketHub
	validate *validator.Validate
}

func NewHandler(service chatMessageService, hub webSocketHub) *Handler {
	return &Handler{service: service, hub: hub, validate: validator.New()}
}

func (h *Handler) Create(c *gin.Context) {
	roomID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid room ID")
		return
	}

	var req createMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.validate.Struct(req); err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	userID, _ := current.UserID(c)
	msg, err := h.service.Create(&CreateChatMessageParams{
		RoomID:   uint(roomID),
		UserID:   userID,
		Content:  req.Content,
		IsPinned: req.IsPinned,
	})
	if err != nil {
		if errors.Is(err, httpx.ErrForbidden) {
			httpx.ErrorResponse(c, http.StatusForbidden, "You don't have permission to send messages")
			return
		}
		httpx.ErrorResponse(c, http.StatusInternalServerError, "Failed to send message")
		return
	}

	h.hub.BroadcastToRoom(uint(roomID), ws.EventChatMessage, BuildChatMessageResponse(msg))
	httpx.SuccessResponse(c, http.StatusCreated, "Message sent successfully", BuildChatMessageResponse(msg))
}

func (h *Handler) List(c *gin.Context) {
	roomID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid room ID")
		return
	}

	var filter ChatMessageFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid filter")
		return
	}
	filter.RoomID = uint(roomID)

	var p listopts.Pagination
	if err := c.ShouldBindQuery(&p); err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid pagination params")
		return
	}
	if p.Page <= 0 || p.PageSize <= 0 || p.PageSize > 50 {
		httpx.ErrorResponse(c, http.StatusBadRequest, "page and pageSize must be positive and pageSize must be less than 50")
		return
	}

	userID, _ := current.UserID(c)
	messages, count, err := h.service.List(userID, filter, p)
	if err != nil {
		if errors.Is(err, httpx.ErrForbidden) {
			httpx.ErrorResponse(c, http.StatusForbidden, "You are not a member of this room")
			return
		}
		httpx.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch messages")
		return
	}

	httpx.PaginatedSuccessResponse(c, "Messages fetched successfully", BuildChatMessageListResponse(messages), p.Page, p.PageSize, int(count))
}

func (h *Handler) SetPinned(c *gin.Context) {
	roomID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid room ID")
		return
	}

	messageID, err := strconv.Atoi(c.Param("messageId"))
	if err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid message ID")
		return
	}

	var req setPinnedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	userID, _ := current.UserID(c)
	msg, err := h.service.SetPinned(userID, uint(roomID), uint(messageID), req.IsPinned)
	if err != nil {
		switch {
		case errors.Is(err, httpx.ErrForbidden):
			httpx.ErrorResponse(c, http.StatusForbidden, "You don't have permission to pin messages")
		case errors.Is(err, httpx.ErrRecordNotFound):
			httpx.ErrorResponse(c, http.StatusNotFound, "Message not found")
		case errors.Is(err, httpx.ErrPinnedLimitReached):
			httpx.ErrorResponse(c, http.StatusConflict, "Pinned messages limit reached")
		default:
			httpx.ErrorResponse(c, http.StatusInternalServerError, "Failed to update message")
		}
		return
	}

	httpx.SuccessResponse(c, http.StatusOK, "Message updated successfully", BuildChatMessageResponse(msg))
}
