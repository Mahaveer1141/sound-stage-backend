package roomuser

import (
	"context"
	"errors"
	"net/http"
	"sound-stage-backend/internal/pkg/current"
	"sound-stage-backend/internal/pkg/httpx"
	"sound-stage-backend/internal/pkg/listopts"
	"sound-stage-backend/internal/role"
	"sound-stage-backend/internal/ws"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type roomUserService interface {
	ListByRoomID(ctx context.Context, roomID, userID uint, filter RoomUserFilter, c listopts.Cursor) ([]RoomUser, int64, bool, string, error)
	FindByWithState(ctx context.Context, userID, roomID uint) (*RoomUser, error)
	ListRaisedHands(ctx context.Context, roomID, userID uint, p listopts.Pagination) ([]RoomUser, int64, error)
	UpdateRole(ctx context.Context, roomID, userID uint, roleName role.RoleName, actorID uint) (*RoomUser, error)
	DeleteUser(ctx context.Context, roomID, userID, actorID uint) (*RoomUser, error)
	Block(roomID, userID, actorID uint) (*RoomUser, error)
	CountsByRoomID(roomID uint) (RoomUserCounts, error)
	Unblock(roomID, userID, actorID uint) error
	ListBlockedByRoomID(roomID, actorID uint, filter RoomUserFilter, p listopts.Pagination) ([]RoomUser, int64, error)
	SetMuted(ctx context.Context, roomID, userID, actorID uint, isMuted bool) error
}

type webSocketBroadcaster interface {
	BroadcastToRoom(roomID uint, eventName ws.EventName, payload any)
}

type roomJoiner interface {
	AddRoomUser(roomID, userID uint, privateCode string) (*RoomUser, error)
}

type Handler struct {
	service  roomUserService
	joiner   roomJoiner
	validate *validator.Validate
	hub      webSocketBroadcaster
}

func NewHandler(service roomUserService, joiner roomJoiner, hub webSocketBroadcaster) *Handler {
	return &Handler{service: service, joiner: joiner, validate: validator.New(), hub: hub}
}

type updateUserRoleInput struct {
	Role role.RoleName `json:"role" validate:"required"`
}

type addRoomUserInput struct {
	PrivateCode string `json:"privateCode"`
}

func (h *Handler) ListUsers(c *gin.Context) {
	id := c.Param("id")
	roomId, err := strconv.Atoi(id)
	if err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid room ID")
		return
	}

	var cur listopts.Cursor
	if err := c.ShouldBindQuery(&cur); err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid cursor params")
		return
	}
	if cur.Limit <= 0 {
		cur.Limit = 20
	}
	if cur.Limit > 50 {
		httpx.ErrorResponse(c, http.StatusBadRequest, "limit must not exceed 50")
		return
	}

	var filter RoomUserFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid filter")
		return
	}

	currentUserID, _ := current.UserID(c)
	users, count, hasMore, nextCursor, err := h.service.ListByRoomID(c.Request.Context(), uint(roomId), currentUserID, filter, cur)
	if err != nil {
		if errors.Is(err, httpx.ErrUserBlocked) {
			httpx.ErrorResponse(c, http.StatusForbidden, "You are blocked from this room")
			return
		}

		httpx.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch room users")
		return
	}

	userResponses := BuildRoomUserListResponse(users, currentUserID)

	httpx.CursorPaginatedSuccessResponse(c, "Room users fetched successfully", userResponses, cur.Limit, count, nextCursor, hasMore)
}

func (h *Handler) CurrentRoomUser(c *gin.Context) {
	id := c.Param("id")
	roomId, err := strconv.Atoi(id)
	if err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid room ID")
		return
	}
	userID, _ := current.UserID(c)

	ru, err := h.service.FindByWithState(c.Request.Context(), userID, uint(roomId))
	if err != nil || ru == nil {
		httpx.ErrorResponse(c, http.StatusUnprocessableEntity, "Failed to fetch user")
		return
	}
	httpx.SuccessResponse(c, http.StatusOK, "Successfully fetch the user", BuildRoomUserResponse(ru, userID))
}

func (h *Handler) AddRoomUser(c *gin.Context) {
	id := c.Param("id")
	roomId, err := strconv.Atoi(id)
	if err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid room ID")
		return
	}

	userID, _ := current.UserID(c)

	var input addRoomUserInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid request payload")
		return
	}

	ru, err := h.joiner.AddRoomUser(uint(roomId), userID, input.PrivateCode)
	if err != nil {
		if errors.Is(err, httpx.ErrUserBlocked) {
			httpx.ErrorResponse(c, http.StatusForbidden, "You are blocked from this room")
			return
		}
		if errors.Is(err, httpx.ErrForbidden) {
			httpx.ErrorResponse(c, http.StatusForbidden, "Incorrect Code")
			return
		}
		httpx.ErrorResponse(c, http.StatusUnprocessableEntity, "Failed to add user to room")
		return
	}

	httpx.SuccessResponse(c, http.StatusOK, "User added to room", BuildRoomUserResponse(ru, userID))
}

func (h *Handler) UpdateUserRole(c *gin.Context) {
	id := c.Param("id")
	roomID, err := strconv.Atoi(id)
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
	actorID, _ := current.UserID(c)

	var input updateUserRoleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if err := h.validate.Struct(input); err != nil {
		httpx.ErrorResponse(c, http.StatusUnprocessableEntity, "Validation error: "+err.Error())
		return
	}

	ru, err := h.service.UpdateRole(c.Request.Context(), uint(roomID), uint(userID), input.Role, actorID)
	if err != nil {
		if errors.Is(err, httpx.ErrForbidden) {
			httpx.ErrorResponse(c, http.StatusForbidden, httpx.ErrForbidden.Error())
			return
		}
		if errors.Is(err, httpx.ErrRecordNotFound) {
			httpx.ErrorResponse(c, http.StatusNotFound, "User not found in room")
			return
		}
		httpx.ErrorResponse(c, http.StatusUnprocessableEntity, "Failed to update user role")
		return
	}

	counts, err := h.service.CountsByRoomID(uint(roomID))
	if err != nil {
		httpx.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch room counts")
		return
	}

	h.hub.BroadcastToRoom(uint(roomID), ws.EventUserRoleUpdated, RoomUserEventPayload{
		RoomUserCounts: counts,
		RoomUser:       BuildRoomUserResponse(ru, 0),
	})

	httpx.SuccessResponse(c, http.StatusOK, "User role updated successfully", BuildRoomUserResponse(ru, actorID))
}

func (h *Handler) DeleteUser(c *gin.Context) {
	id := c.Param("id")
	roomID, err := strconv.Atoi(id)
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
	actorID, _ := current.UserID(c)

	ru, err := h.service.DeleteUser(c.Request.Context(), uint(roomID), uint(userID), actorID)
	if err != nil {
		if errors.Is(err, httpx.ErrForbidden) {
			httpx.ErrorResponse(c, http.StatusForbidden, httpx.ErrForbidden.Error())
			return
		}
		if errors.Is(err, httpx.ErrUserBlocked) {
			httpx.ErrorResponse(c, http.StatusForbidden, "Cannot remove a blocked user")
			return
		}
		httpx.ErrorResponse(c, http.StatusUnprocessableEntity, "Failed to remove user from room")
		return
	}

	counts, err := h.service.CountsByRoomID(uint(roomID))
	if err != nil {
		httpx.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch room counts")
		return
	}

	eventName := ws.EventUserKickedOut
	if actorID == uint(userID) {
		eventName = ws.EventLeaveRoom
	}
	h.hub.BroadcastToRoom(uint(roomID), eventName, RoomUserRemovedPayload{
		RoomUserCounts: counts,
		UserID:         uint(userID),
		CanSpeak:       ru.CanSpeak(),
	})

	httpx.SuccessResponse(c, http.StatusOK, "User removed from room", nil)
}

func (h *Handler) SetUserMuted(c *gin.Context) {
	id := c.Param("id")
	roomID, err := strconv.Atoi(id)
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
	actorID, _ := current.UserID(c)

	if err := h.service.SetMuted(c.Request.Context(), uint(roomID), uint(userID), actorID, true); err != nil {
		if errors.Is(err, httpx.ErrForbidden) {
			httpx.ErrorResponse(c, http.StatusForbidden, httpx.ErrForbidden.Error())
			return
		}
		if errors.Is(err, httpx.ErrUserBlocked) {
			httpx.ErrorResponse(c, http.StatusForbidden, "You are blocked from this room")
			return
		}
		if errors.Is(err, httpx.ErrRecordNotFound) {
			httpx.ErrorResponse(c, http.StatusNotFound, "User not found in room")
			return
		}
		httpx.ErrorResponse(c, http.StatusUnprocessableEntity, "Failed to update mute state")
		return
	}

	httpx.SuccessResponse(c, http.StatusOK, "User mute state updated", nil)
}

func (h *Handler) ListRaisedHands(c *gin.Context) {
	id := c.Param("id")
	roomID, err := strconv.Atoi(id)
	if err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid room ID")
		return
	}

	var p listopts.Pagination
	if err := c.ShouldBindQuery(&p); err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid pagination params")
		return
	}
	if p.Page <= 0 || p.PageSize <= 0 || p.PageSize > 50 {
		httpx.ErrorResponse(c, http.StatusBadRequest, "page and pageSize must be positive and pageSize must be less than 50")
		return
	}

	currentUserID, _ := current.UserID(c)
	users, count, err := h.service.ListRaisedHands(c.Request.Context(), uint(roomID), currentUserID, p)
	if err != nil {
		if errors.Is(err, httpx.ErrUserBlocked) {
			httpx.ErrorResponse(c, http.StatusForbidden, "You are blocked from this room")
			return
		}
		httpx.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch raised hands")
		return
	}

	httpx.PaginatedSuccessResponse(c, "Raised hands fetched successfully", BuildRoomUserListResponse(users, currentUserID), p.Page, p.PageSize, int(count))
}

func (h *Handler) ListBlockedUsers(c *gin.Context) {
	id := c.Param("id")
	roomID, err := strconv.Atoi(id)
	if err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid room ID")
		return
	}

	var p listopts.Pagination
	if err := c.ShouldBindQuery(&p); err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid pagination params")
		return
	}
	if p.Page <= 0 || p.PageSize <= 0 || p.PageSize > 50 {
		httpx.ErrorResponse(c, http.StatusBadRequest, "page and pageSize must be positive and pageSize must be less than 50")
		return
	}

	var filter RoomUserFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid filter")
		return
	}

	actorID, _ := current.UserID(c)
	users, count, err := h.service.ListBlockedByRoomID(uint(roomID), actorID, filter, p)
	if err != nil {
		if errors.Is(err, httpx.ErrForbidden) {
			httpx.ErrorResponse(c, http.StatusForbidden, httpx.ErrForbidden.Error())
			return
		}
		httpx.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch blocked users")
		return
	}

	httpx.PaginatedSuccessResponse(c, "Blocked users fetched successfully", BuildBlockedUserListResponse(users), p.Page, p.PageSize, int(count))
}

func (h *Handler) BlockUser(c *gin.Context) {
	id := c.Param("id")
	roomID, err := strconv.Atoi(id)
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

	actorID, _ := current.UserID(c)
	ru, err := h.service.Block(uint(roomID), uint(userID), actorID)
	if err != nil {
		if errors.Is(err, httpx.ErrForbidden) {
			httpx.ErrorResponse(c, http.StatusForbidden, httpx.ErrForbidden.Error())
			return
		}
		if errors.Is(err, httpx.ErrRecordNotFound) {
			httpx.ErrorResponse(c, http.StatusNotFound, "User not found in room")
			return
		}
		httpx.ErrorResponse(c, http.StatusUnprocessableEntity, "Failed to block user")
		return
	}

	counts, err := h.service.CountsByRoomID(uint(roomID))
	if err != nil {
		httpx.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch room counts")
		return
	}

	h.hub.BroadcastToRoom(uint(roomID), ws.EventUserKickedOut, RoomUserRemovedPayload{
		RoomUserCounts: counts,
		UserID:         uint(userID),
		CanSpeak:       ru.CanSpeak(),
	})

	httpx.SuccessResponse(c, http.StatusOK, "User blocked from room", nil)
}

func (h *Handler) UnblockUser(c *gin.Context) {
	id := c.Param("id")
	roomID, err := strconv.Atoi(id)
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
	actorID, _ := current.UserID(c)

	if err := h.service.Unblock(uint(roomID), uint(userID), actorID); err != nil {
		if errors.Is(err, httpx.ErrForbidden) {
			httpx.ErrorResponse(c, http.StatusForbidden, httpx.ErrForbidden.Error())
			return
		}
		if errors.Is(err, httpx.ErrRecordNotFound) {
			httpx.ErrorResponse(c, http.StatusNotFound, "User is not blocked in this room")
			return
		}
		httpx.ErrorResponse(c, http.StatusUnprocessableEntity, "Failed to unblock user")
		return
	}

	httpx.SuccessResponse(c, http.StatusOK, "User unblocked from room", nil)
}
