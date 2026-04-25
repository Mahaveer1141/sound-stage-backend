package room

import (
	"net/http"
	"sound-stage-backend/internal/pkg/httpx"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type Handler interface {
	List(c *gin.Context)
	FindByID(c *gin.Context)
	Create(c *gin.Context)
	Update(c *gin.Context)
}

type handler struct {
	service  Service
	validate *validator.Validate
}

func NewHandler(service Service) Handler {
	return &handler{service: service, validate: validator.New()}
}

func (h *handler) Create(c *gin.Context) {
	userId, _ := c.Get("userId")
	var input CreateRoomParams
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid request payload")
		return
	}
	input.CreatorID = userId.(uint)
	if err := h.validate.Struct(input); err != nil {
		httpx.ErrorResponse(c, http.StatusUnprocessableEntity, "Validation error: "+err.Error())
		return
	}

	room, err := h.service.Create(&input)
	if err != nil {
		httpx.ErrorResponse(c, http.StatusInternalServerError, "Failed to create room")
		return
	}

	httpx.SuccessResponse(c, http.StatusOK, "Room created successfully", room)
}

func (h *handler) Update(c *gin.Context) {
	id := c.Param("id")
	roomId, err := strconv.Atoi(id)
	if err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid room ID")
		return
	}

	var input UpdateRoomParams
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := h.validate.Struct(input); err != nil {
		httpx.ErrorResponse(c, http.StatusUnprocessableEntity, "Validation error: "+err.Error())
		return
	}

	room, err := h.service.Update(uint(roomId), &input)
	if err != nil {
		httpx.ErrorResponse(c, http.StatusInternalServerError, "Failed to update room")
		return
	}

	httpx.SuccessResponse(c, http.StatusOK, "Room updated successfully", room)
}

func (h *handler) List(c *gin.Context) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page <= 0 {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid page")
		return
	}

	pageSize, err := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	if err != nil || pageSize <= 0 {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid pageSize")
		return
	}

	rooms, err := h.service.List(page, pageSize)
	if err != nil {
		httpx.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch rooms")
		return
	}

	count, err := h.service.Count()
	if err != nil {
		httpx.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch rooms count")
		return
	}

	httpx.PaginatedSuccessResponse(c, "Rooms fetched successfully", rooms, page, pageSize, count)
}

func (h *handler) FindByID(c *gin.Context) {
	id := c.Param("id")
	roomId, err := strconv.Atoi(id)
	if err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid room ID")
		return
	}

	room, err := h.service.FindByID(uint(roomId))
	if err != nil {
		httpx.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch room")
		return
	}

	httpx.SuccessResponse(c, http.StatusOK, "Room fetched successfully", room)
}
