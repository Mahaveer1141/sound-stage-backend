package middleware

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func resetBuckets(t *testing.T) {
	t.Helper()
	mu.Lock()
	buckets = make(map[string]*Bucket)
	mu.Unlock()
}

func setupRateLimiterEngine(t *testing.T) (*gin.Engine, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	r := gin.New()
	r.Use(RateLimiter(logger))
	r.Any("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	return r, w
}

func newRateLimitRequest(remoteIP string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = remoteIP + ":12345"
	return req
}

func TestRateLimiter(t *testing.T) {
	t.Run("allows OPTIONS requests", func(t *testing.T) {
		resetBuckets(t)
		r, _ := setupRateLimiterEngine(t)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodOptions, "/test", nil)
		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("allows requests within token limit", func(t *testing.T) {
		resetBuckets(t)
		r, _ := setupRateLimiterEngine(t)

		ip := "1.2.3.4"
		for range MAX_TOKENS {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, newRateLimitRequest(ip))
			require.Equal(t, http.StatusOK, w.Code)
		}
	})

	t.Run("rejects requests when tokens are exhausted", func(t *testing.T) {
		resetBuckets(t)
		r, _ := setupRateLimiterEngine(t)

		ip := "1.2.3.4"
		for range MAX_TOKENS {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, newRateLimitRequest(ip))
			require.Equal(t, http.StatusOK, w.Code)
		}

		w := httptest.NewRecorder()
		r.ServeHTTP(w, newRateLimitRequest(ip))
		require.Equal(t, http.StatusTooManyRequests, w.Code)
	})

	t.Run("refills tokens over time", func(t *testing.T) {
		resetBuckets(t)
		r, _ := setupRateLimiterEngine(t)

		ip := "1.2.3.4"
		for range MAX_TOKENS {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, newRateLimitRequest(ip))
			require.Equal(t, http.StatusOK, w.Code)
		}

		w := httptest.NewRecorder()
		r.ServeHTTP(w, newRateLimitRequest(ip))
		require.Equal(t, http.StatusTooManyRequests, w.Code)

		// 100 tokens per minute ~= 1 token every 600ms
		time.Sleep(700 * time.Millisecond)

		w = httptest.NewRecorder()
		r.ServeHTTP(w, newRateLimitRequest(ip))
		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("tracks clients independently by IP", func(t *testing.T) {
		resetBuckets(t)
		r, _ := setupRateLimiterEngine(t)

		firstIP := "1.2.3.4"
		secondIP := "2.2.2.2"

		for range MAX_TOKENS {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, newRateLimitRequest(firstIP))
			require.Equal(t, http.StatusOK, w.Code)
		}

		w := httptest.NewRecorder()
		r.ServeHTTP(w, newRateLimitRequest(firstIP))
		require.Equal(t, http.StatusTooManyRequests, w.Code)

		w = httptest.NewRecorder()
		r.ServeHTTP(w, newRateLimitRequest(secondIP))
		require.Equal(t, http.StatusOK, w.Code)
	})
}
