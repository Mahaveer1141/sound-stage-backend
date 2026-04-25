package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"sound-stage-backend/internal/pkg/httpx"

	"github.com/gin-gonic/gin"
)

func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				requestID, _ := c.Get("request_id")
				logger.Error("Panic recovered",
					slog.Any("error", err),
					slog.String("request_id", requestID.(string)),
					slog.String("path", c.Request.URL.Path),
					slog.String("method", c.Request.Method),
					slog.String("ip_address", c.ClientIP()),
					slog.String("stack", string(debug.Stack())),
				)

				httpx.ErrorResponse(c, http.StatusInternalServerError, "an unexpected error occurred")
				c.Abort()
			}
		}()
		c.Next()
	}
}
