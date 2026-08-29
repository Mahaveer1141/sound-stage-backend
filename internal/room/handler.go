package room

import (
	"errors"
	"net/http"
	"sound-stage-backend/internal/pkg/current"
	"sound-stage-backend/internal/pkg/httpx"
	"sound-stage-backend/internal/pkg/listopts"
	"sound-stage-backend/internal/role"
	roomuser "sound-stage-backend/internal/room_user"
	"sound-stage-backend/internal/ws"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type roomService interface {
	List(filter RoomFilter, sort listopts.Sort, p listopts.Pagination) ([]Room, int64, error)
	FindByID(id uint) (*Room, error)
	Create(input *CreateRoomParams) (*Room, error)
	Update(id, userID uint, input *UpdateRoomParams) (*Room, error)
	ListUsers(roomID uint, filter roomuser.RoomUserFilter, sort listopts.Sort, p listopts.Pagination) ([]roomuser.RoomUser, int64, error)
	UpdateUserRole(roomID uint, userID uint, newRole role.RoleName, actorID uint) error
	CurrentRoomUser(roomID uint, userID uint) (*roomuser.RoomUser, error)
	UpdatePrivateCode(roomID uint) error
	AddRoomUser(roomID, userID uint, privateCode string) (*roomuser.RoomUser, error)
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
	return &Handler{service: service, validate: validator.New(), hub: hub}
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

func (h *Handler) ListUsers(c *gin.Context) {
	id := c.Param("id")
	roomId, err := strconv.Atoi(id)
	if err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid room ID")
		return
	}

	var p listopts.Pagination
	if err := c.ShouldBindQuery(&p); err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid pagination params")
		return
	}
	if p.Page <= 0 || p.PageSize <= 0 {
		httpx.ErrorResponse(c, http.StatusBadRequest, "page and pageSize must be positive")
		return
	}

	var filter roomuser.RoomUserFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid filter")
		return
	}

	var sort listopts.Sort
	if err := c.ShouldBindQuery(&sort); err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid sort")
		return
	}

	users, count, err := h.service.ListUsers(uint(roomId), filter, sort, p)
	if err != nil {
		httpx.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch room users")
		return
	}

	currentUserID, _ := current.UserID(c)
	viewerRoomUser, _ := h.service.CurrentRoomUser(uint(roomId), currentUserID)
	viewerIsAdmin := viewerRoomUser != nil && viewerRoomUser.IsAdmin()

	userResponses := roomuser.BuildRoomUserListResponse(users, currentUserID, viewerIsAdmin)

	httpx.PaginatedSuccessResponse(c, "Room users fetched successfully", userResponses, p.Page, p.PageSize, int(count))
}

func (h *Handler) UpdateUserRole(c *gin.Context) {
	id := c.Param("id")
	roomID, err := strconv.Atoi(id)
	userId := c.Param("userId")
	userID, err := strconv.Atoi(userId)
	if err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid ID's")
		return
	}
	actorId, _ := c.Get("userId")
	actorID, _ := actorId.(uint)

	var input UpdateUserRoleParams
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if err := h.validate.Struct(input); err != nil {
		httpx.ErrorResponse(c, http.StatusUnprocessableEntity, "Validation error: "+err.Error())
		return
	}

	if err := h.service.UpdateUserRole(uint(roomID), uint(userID), input.Role, actorID); err != nil {
		if errors.Is(err, httpx.ErrForbidden) {
			httpx.ErrorResponse(c, http.StatusForbidden, httpx.ErrForbidden.Error())
			return
		}
		httpx.ErrorResponse(c, http.StatusUnprocessableEntity, "Failed to update user role")
		return
	}

	h.hub.BroadcastToRoom(uint(roomID), ws.EventUserRoleUpdated, gin.H{"userId": userID, "role": input.Role})

	httpx.SuccessResponse(c, http.StatusOK, "User role updated successfully", nil)
}

func (h *Handler) CurrentRoomUser(c *gin.Context) {
	id := c.Param("id")
	roomId, err := strconv.Atoi(id)
	if err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid room ID")
		return
	}
	userID, _ := current.UserID(c)

	ru, err := h.service.CurrentRoomUser(uint(roomId), userID)
	if err != nil || ru == nil {
		httpx.ErrorResponse(c, http.StatusUnprocessableEntity, "Failed to fetch user")
		return
	}
	viewerIsAdmin := ru.IsAdmin()
	httpx.SuccessResponse(c, http.StatusOK, "Successfully fetch the user", roomuser.BuildRoomUserResponse(ru, userID, viewerIsAdmin))
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

func (h *Handler) AddRoomUser(c *gin.Context) {
	id := c.Param("id")
	roomId, err := strconv.Atoi(id)
	if err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid room ID")
		return
	}

	userID, _ := current.UserID(c)

	var input AddRoomUserInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid request payload")
		return
	}

	ru, err := h.service.AddRoomUser(uint(roomId), userID, input.PrivateCode)
	if err != nil {
		if errors.Is(err, httpx.ErrForbidden) {
			httpx.ErrorResponse(c, http.StatusForbidden, "Incorrect Code")
			return
		}
		httpx.ErrorResponse(c, http.StatusUnprocessableEntity, "Failed to add user to room")
		return
	}

	viewerIsAdmin := ru.IsAdmin()
	httpx.SuccessResponse(c, http.StatusOK, "User added to room", roomuser.BuildRoomUserResponse(ru, userID, viewerIsAdmin))
}
