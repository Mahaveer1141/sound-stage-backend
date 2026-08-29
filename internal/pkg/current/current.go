package current

import (
	"context"

	"github.com/gin-gonic/gin"
)

type userIDKey struct{}

func UserIDFromContext(ctx context.Context) (uint, bool) {
	id, ok := ctx.Value(userIDKey{}).(uint)
	return id, ok
}

func WithUserID(ctx context.Context, userID uint) context.Context {
	return context.WithValue(ctx, userIDKey{}, userID)
}

func Set(c *gin.Context, userID uint) {
	c.Set("userId", userID)
	c.Request = c.Request.WithContext(WithUserID(c.Request.Context(), userID))
}

func UserID(c *gin.Context) (uint, bool) {
	v, ok := c.Get("userId")
	if !ok {
		return 0, false
	}
	id, ok := v.(uint)
	return id, ok
}
