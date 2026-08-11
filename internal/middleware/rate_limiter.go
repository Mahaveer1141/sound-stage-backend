package middleware

import (
	"log/slog"
	"net/http"
	"sound-stage-backend/internal/pkg/httpx"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type Bucket struct {
	AvailableTokens float64
	LastRefreshedAt time.Time
}

const MAX_TOKENS = 100
const REFILL_RATE = float64(MAX_TOKENS) / float64(time.Minute)

var (
	buckets = make(map[string]*Bucket)
	mu      sync.Mutex
)

func RateLimiter(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		clientIP := c.ClientIP()

		mu.Lock()

		bucket, exists := buckets[clientIP]
		if !exists {
			bucket = &Bucket{
				AvailableTokens: MAX_TOKENS,
				LastRefreshedAt: time.Now(),
			}
			buckets[clientIP] = bucket
		}

		now := time.Now()
		elapsed := now.Sub(bucket.LastRefreshedAt)
		refill := float64(elapsed) * REFILL_RATE

		bucket.AvailableTokens = min(float64(MAX_TOKENS), bucket.AvailableTokens+refill)
		bucket.LastRefreshedAt = now

		if bucket.AvailableTokens < 1 {
			httpx.ErrorResponse(c, http.StatusTooManyRequests, "rate limit exceeded")
			logger.Error("rate limit exceeded", "ip", clientIP)
			mu.Unlock()
			c.Abort()
			return
		}

		bucket.AvailableTokens--
		mu.Unlock()

		c.Next()
	}
}

func StartBucketCleanup() {
	for {
		time.Sleep(10 * time.Minute)
		mu.Lock()
		for ip, b := range buckets {
			if time.Since(b.LastRefreshedAt) > 30*time.Minute {
				delete(buckets, ip)
			}
		}
		mu.Unlock()
	}
}
