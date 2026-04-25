package middleware

import (
	"net/http"
	apitoken "sound-stage-backend/internal/api_token"
	"sound-stage-backend/internal/pkg/httpx"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(apiTokenService apitoken.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		if tokenString == "" {
			httpx.ErrorResponse(c, http.StatusUnauthorized, "Unauthorized")
			c.Abort()
			return
		}

		userID, err := apiTokenService.ValidateToken(tokenString, apitoken.AccessToken)
		if err != nil {
			httpx.ErrorResponse(c, http.StatusUnauthorized, err.Error())
			c.Abort()
			return
		}

		c.Set("userId", userID)
		c.Next()
	}
}
