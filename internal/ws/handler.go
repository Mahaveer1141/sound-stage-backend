package ws

import (
	"net/http"
	"sound-stage-backend/internal/config"
	"sound-stage-backend/internal/pkg/current"
	"sound-stage-backend/internal/pkg/httpx"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type EventHandler func(c *Client, evt Event)

type DisconnectHandler func(c *Client)

type Handler interface {
	On(en EventName, handler EventHandler)
	OnDisconnect(handler DisconnectHandler)
	ServeWS(ctx *gin.Context)
}

type JoinValidator func(roomID, userID uint) error

type handler struct {
	hub         *Hub
	handlers    map[EventName]EventHandler
	disconnects []DisconnectHandler
	cfg         *config.Config
	canJoin     JoinValidator
}

func NewHandler(hub *Hub, cfg *config.Config, canJoin JoinValidator) Handler {
	return &handler{
		hub:      hub,
		handlers: make(map[EventName]EventHandler),
		cfg:      cfg,
		canJoin:  canJoin,
	}
}

func (h *handler) On(en EventName, handler EventHandler) {
	h.handlers[en] = handler
}

func (h *handler) OnDisconnect(handler DisconnectHandler) {
	h.disconnects = append(h.disconnects, handler)
}

func (h *handler) ServeWS(ctx *gin.Context) {
	roomID := ctx.Param("roomId")
	userID, _ := current.UserID(ctx)
	parsedRoomID, err := strconv.ParseUint(roomID, 10, 0)
	if err != nil {
		httpx.ErrorResponse(ctx, http.StatusBadRequest, "Invalid room ID")
		return
	}

	if err := h.canJoin(uint(parsedRoomID), userID); err != nil {
		httpx.ErrorResponse(ctx, http.StatusForbidden, "You are not a member of this room")
		return
	}

	var upgrader = websocket.Upgrader{
		ReadBufferSize:  h.cfg.WebSocket.ReadBufferSize,
		WriteBufferSize: h.cfg.WebSocket.WriteBufferSize,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}

	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		httpx.ErrorResponse(ctx, http.StatusInternalServerError, "Failed to upgrade to WebSocket")
		return
	}
	client := newClient(uint(parsedRoomID), userID, h.hub, conn, h.cfg)
	h.hub.register <- client

	go client.writePump()
	client.readPump(h.handleEvent)

	for _, fn := range h.disconnects {
		fn(client)
	}
}

func (h *handler) handleEvent(c *Client, evt Event) {
	fn, ok := h.handlers[evt.Name]
	if ok {
		fn(c, evt)
	}
}
