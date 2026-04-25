package health

import (
	"net/http"
	"sound-stage-backend/internal/pkg/httpx"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handler interface {
	Health(c *gin.Context)
}

type handler struct {
	db *gorm.DB
}

func NewHandler(db *gorm.DB) Handler {
	return &handler{db: db}
}

func (h *handler) Health(c *gin.Context) {
	sqlDB, err := h.db.DB()

	if err != nil || sqlDB.Ping() != nil {
		httpx.ErrorResponse(
			c,
			http.StatusInternalServerError,
			"database connection failed")
		return
	}

	httpx.SuccessResponse(
		c,
		http.StatusOK,
		"service is healthy",
		nil,
	)
}
