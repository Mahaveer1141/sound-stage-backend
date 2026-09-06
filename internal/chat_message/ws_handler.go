package chatmessage

import (
	"encoding/json"
	"errors"
	"net/http"

	"sound-stage-backend/internal/pkg/httpx"
	"sound-stage-backend/internal/ws"
)

type createChatMessagePayload struct {
	Content  string `json:"content"`
	IsPinned bool   `json:"isPinned"`
}

type webSocketHub interface {
	BroadcastToRoom(roomID uint, eventName ws.EventName, payload any)
	ErrorToClient(c *ws.Client, message string, statusCode int)
}

type chatMessageServiceWS interface {
	Create(input *CreateChatMessageParams) (*ChatMessage, error)
}

type WsHandler struct {
	service chatMessageServiceWS
	hub     webSocketHub
}

func NewWSHandler(service chatMessageServiceWS, hub webSocketHub) *WsHandler {
	return &WsHandler{service: service, hub: hub}
}

func (h *WsHandler) Register(wsh ws.Handler) {
	wsh.On(ws.EventCreateChatMessage, h.handleCreateChatMessage)
}

func (h *WsHandler) handleCreateChatMessage(c *ws.Client, evt ws.Event) {
	var p createChatMessagePayload
	if err := json.Unmarshal(evt.Payload, &p); err != nil {
		h.hub.ErrorToClient(c, "Invalid chat message payload", http.StatusUnprocessableEntity)
		return
	}

	msg, err := h.service.Create(&CreateChatMessageParams{
		RoomID:   c.RoomID,
		UserID:   c.UserID,
		Content:  p.Content,
		IsPinned: p.IsPinned,
	})
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, httpx.ErrForbidden) {
			status = http.StatusForbidden
		}
		h.hub.ErrorToClient(c, "Failed to create chat message", status)
		return
	}

	h.hub.BroadcastToRoom(c.RoomID, ws.EventChatMessage, BuildChatMessageResponse(msg))
}
