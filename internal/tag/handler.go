package tag

import (
	"net/http"

	"sound-stage-backend/internal/pkg/httpx"
	"sound-stage-backend/internal/pkg/listopts"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type tagService interface {
	Create(input *CreateTagParams) (*Tag, error)
	List(filter TagFilter, sort listopts.Sort, p listopts.Pagination) ([]Tag, int64, error)
}

type Handler struct {
	service  tagService
	validate *validator.Validate
}

func NewHandler(service tagService) *Handler {
	return &Handler{service: service, validate: validator.New()}
}

func (h *Handler) Create(c *gin.Context) {
	var input CreateTagParams
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if err := h.validate.Struct(input); err != nil {
		httpx.ErrorResponse(c, http.StatusUnprocessableEntity, "Validation error: "+err.Error())
		return
	}

	t, err := h.service.Create(&input)
	if err != nil {
		httpx.ErrorResponse(c, http.StatusUnprocessableEntity, "Failed to create tag")
		return
	}

	httpx.SuccessResponse(c, http.StatusOK, "Tag created successfully", BuildTagResponse(t))
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

	var filter TagFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid filter")
		return
	}

	var sort listopts.Sort
	if err := c.ShouldBindQuery(&sort); err != nil {
		httpx.ErrorResponse(c, http.StatusBadRequest, "Invalid sort")
		return
	}

	tags, count, err := h.service.List(filter, sort, p)
	if err != nil {
		httpx.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch tags")
		return
	}

	responses := BuildTagListResponse(tags)

	httpx.PaginatedSuccessResponse(c, "Tags fetched successfully", responses, p.Page, p.PageSize, int(count))
}
