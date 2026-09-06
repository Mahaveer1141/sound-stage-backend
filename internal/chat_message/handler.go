package chatmessage

import (
	"errors"
	"net/http"
	"strconv"

	"sound-stage-backend/internal/pkg/current"
	"sound-stage-backend/internal/pkg/httpx"
	"sound-stage-backend/internal/pkg/listopts"

	"github.com/gin-gonic/gin"
)

type chatMessageService interface {
	List(userID uint, filter ChatMessageFilter, p listopts.Pagination) ([]ChatMessage, int64, error)
}

type Handler struct {
	service chatMessageService
}

func NewHandler(service chatMessageService) *Handler {
	return &Handler{service: service}
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
