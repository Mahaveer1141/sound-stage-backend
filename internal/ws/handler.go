package ws

import (
	"net/http"
	"sound-stage-backend/internal/pkg/httpx"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type EventHandler func(c *Client, evt Event)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type Handler interface {
	On(en EventName, handler EventHandler)
	ServeWS(ctx *gin.Context)
}

type handler struct {
	hub      *Hub
	handlers map[EventName]EventHandler
}

func NewHandler(hub *Hub) Handler {
	return &handler{
		hub:      hub,
		handlers: make(map[EventName]EventHandler),
	}
}

func (h *handler) On(en EventName, handler EventHandler) {
	h.handlers[en] = handler
}

func (h *handler) ServeWS(ctx *gin.Context) {
	roomID := ctx.Param("roomId")
	userID, _ := ctx.Get("userId")
	parsedRoomID, err := strconv.ParseUint(roomID, 10, 0)
	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		httpx.ErrorResponse(ctx, http.StatusInternalServerError, "Failed to upgrade to WebSocket")
		return
	}
	client := newClient(uint(parsedRoomID), userID.(uint), h.hub, conn)
	h.hub.register <- client

	go client.writePump()
	client.readPump(h.handleEvent)
}

func (h *handler) handleEvent(c *Client, evt Event) {
	fn, ok := h.handlers[evt.Name]
	if ok {
		fn(c, evt)
	}
}
