package room

import (
	roomuser "sound-stage-backend/internal/room_user"
	"sound-stage-backend/internal/ws"
)

type WSHandler interface {
	Register(wsh ws.Handler)
	handleUserJoined(c *ws.Client, evt ws.Event)
	handleUserLeft(c *ws.Client, evt ws.Event)
}

type wsHandler struct {
	hub             *ws.Hub
	roomUserService roomuser.Service
}

func NewWSHandler(hub *ws.Hub, roomUserService roomuser.Service) WSHandler {
	return &wsHandler{
		hub:             hub,
		roomUserService: roomUserService,
	}
}

func (h *wsHandler) Register(wsh ws.Handler) {
	wsh.On(ws.EventJoinRoom, h.handleUserJoined)
	wsh.On(ws.EventLeaveRoom, h.handleUserLeft)
}

func (h *wsHandler) handleUserJoined(c *ws.Client, evt ws.Event) {
	ru, err := h.roomUserService.AddUser(c.UserID, c.RoomID)
	if err != nil {

	}
	h.hub.BroadcastToRoom(c.RoomID, "join_room", ru)
}

func (h *wsHandler) handleUserLeft(c *ws.Client, evt ws.Event) {
	h.roomUserService.RemoveUser(c.UserID, c.RoomID)
	h.hub.BroadcastToRoom(c.RoomID, "leave_room", nil)
}
