package category

import (
	"net/http"

	"sound-stage-backend/internal/pkg/httpx"

	"github.com/gin-gonic/gin"
)

type categoryService interface {
	List() ([]Category, error)
}

type Handler struct {
	service categoryService
}

func NewHandler(service categoryService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(c *gin.Context) {
	categories, err := h.service.List()
	if err != nil {
		httpx.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch categories")
		return
	}

	httpx.SuccessResponse(c, http.StatusOK, "Categories fetched successfully", categories)
}
