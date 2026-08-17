package ws

import (
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"

	"sound-stage-backend/internal/config"
)

func newTestWSConfig() *config.Config {
	return &config.Config{
		WebSocket: config.WebSocketConfig{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			MaxMessageSize:  4096,
			PongWait:        time.Hour,
			PingInterval:    time.Hour,
			WriteWait:       time.Second,
		},
	}
}

func newTestHub(t *testing.T) *Hub {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	hub := NewHub(logger)
	go hub.Run()
	return hub
}

func TestHandler_On(t *testing.T) {
	t.Run("registers and invokes an event handler", func(t *testing.T) {
		hub := newTestHub(t)
		handler := NewHandler(hub, &config.Config{}).(*handler)

		called := false
		handler.On(EventJoinRoom, func(c *Client, evt Event) {
			called = true
			require.Equal(t, EventJoinRoom, evt.Name)
		})

		handler.handleEvent(&Client{}, Event{Name: EventJoinRoom})

		require.True(t, called)
	})

	t.Run("unknown event is a no-op", func(t *testing.T) {
		hub := newTestHub(t)
		handler := NewHandler(hub, &config.Config{}).(*handler)

		called := false
		handler.On(EventLeaveRoom, func(c *Client, evt Event) { called = true })

		handler.handleEvent(&Client{}, Event{Name: EventJoinRoom})

		require.False(t, called)
	})

	t.Run("overwrites an existing handler", func(t *testing.T) {
		hub := newTestHub(t)
		handler := NewHandler(hub, &config.Config{}).(*handler)

		first := false
		second := false
		handler.On(EventJoinRoom, func(c *Client, evt Event) { first = true })
		handler.On(EventJoinRoom, func(c *Client, evt Event) { second = true })

		handler.handleEvent(&Client{}, Event{Name: EventJoinRoom})

		require.False(t, first)
		require.True(t, second)
	})
}

func TestHandler_ServeWS(t *testing.T) {
	t.Run("upgrades, calls registered event handler and disconnect handler", func(t *testing.T) {
		hub := newTestHub(t)
		h := NewHandler(hub, newTestWSConfig())

		eventCalled := make(chan *Client, 1)
		disconnectCalled := make(chan *Client, 1)

		h.On(EventJoinRoom, func(c *Client, evt Event) {
			eventCalled <- c
		})
		h.OnDisconnect(func(c *Client) {
			disconnectCalled <- c
		})

		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.GET("/ws/:roomId", func(ctx *gin.Context) {
			ctx.Set("userId", uint(42))
			h.ServeWS(ctx)
		})

		server := httptest.NewServer(r)
		defer server.Close()

		url := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/4"
		ws, _, err := websocket.DefaultDialer.Dial(url, nil)
		require.NoError(t, err)
		defer ws.Close()

		msg, err := Encode(EventJoinRoom, map[string]any{})
		require.NoError(t, err)
		require.NoError(t, ws.WriteMessage(websocket.TextMessage, msg))

		select {
		case c := <-eventCalled:
			require.Equal(t, uint(4), c.RoomID)
			require.Equal(t, uint(42), c.UserID)
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for event handler")
		}

		require.NoError(t, ws.Close())

		select {
		case c := <-disconnectCalled:
			require.Equal(t, uint(4), c.RoomID)
			require.Equal(t, uint(42), c.UserID)
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for disconnect handler")
		}
	})

}
