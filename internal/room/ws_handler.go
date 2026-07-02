package room

import (
	"encoding/json"
	"sound-stage-backend/internal/config"
	roomuser "sound-stage-backend/internal/room_user"
	webrtc "sound-stage-backend/internal/web_rtc"
	"sound-stage-backend/internal/ws"

	pion "github.com/pion/webrtc/v4"
)

type WSHandler interface {
	Register(wsh ws.Handler)
	handleUserJoined(c *ws.Client, evt ws.Event)
	handleUserLeft(c *ws.Client, evt ws.Event)
	handleWebRTCOffer(c *ws.Client, evt ws.Event)
	handleWebRTCCandidate(c *ws.Client, evt ws.Event)
	handleWebRTCAnswer(c *ws.Client, evt ws.Event)
}

type wsHandler struct {
	hub             *ws.Hub
	roomUserService roomuser.Service
	turn            config.TurnConfig
}

func NewWSHandler(hub *ws.Hub, roomUserService roomuser.Service, turn config.TurnConfig) WSHandler {
	return &wsHandler{
		hub:             hub,
		roomUserService: roomUserService,
		turn:            turn,
	}
}

func (h *wsHandler) Register(wsh ws.Handler) {
	wsh.On(ws.EventJoinRoom, h.handleUserJoined)
	wsh.On(ws.EventLeaveRoom, h.handleUserLeft)

	wsh.On(ws.EventWebRTCOffer, h.handleWebRTCOffer)
	wsh.On(ws.EventWebRTCCandidate, h.handleWebRTCCandidate)
	wsh.On(ws.EventWebRTCAnswer, h.handleWebRTCAnswer)
}

func (h *wsHandler) handleUserJoined(c *ws.Client, evt ws.Event) {
	ru, err := h.roomUserService.AddUser(c.UserID, c.RoomID)
	if err != nil {
	}

	pc, err := webrtc.NewPeerConnection(
		h.turn,
		func(ice pion.ICECandidateInit) {
			h.hub.SendToClient(c, ws.EventWebRTCCandidate, ice)
		},
		func(sd pion.SessionDescription) {
			h.hub.SendToClient(c, ws.EventWebRTCOffer, sd)
		})

	c.PC = pc

	pc.OnTrack(func(tr *pion.TrackRemote, r *pion.RTPReceiver) {
		localTrack, _ := webrtc.NewForwardingTrack(tr)
		stop := make(chan struct{})
		go webrtc.ForwardRTP(tr, localTrack, stop)
		h.hub.ForEachClientInRoom(c.RoomID, func(client *ws.Client) {
			if client != c {
				webrtc.AddTrack(client.PC, localTrack)
			}
		})
	})

	h.hub.BroadcastToRoom(c.RoomID, ws.EventJoinRoom, ru)
}

func (h *wsHandler) handleUserLeft(c *ws.Client, evt ws.Event) {
	h.roomUserService.RemoveUser(c.UserID, c.RoomID)
	h.hub.BroadcastToRoom(c.RoomID, ws.EventLeaveRoom, nil)
}

func (h *wsHandler) handleWebRTCOffer(c *ws.Client, evt ws.Event) {
	if c.PC == nil {
		return
	}
	var offer pion.SessionDescription
	err := json.Unmarshal(evt.Payload, &offer)
	if err != nil {
	}
	answer, err := webrtc.HandleOffer(c.PC, offer)
	if err != nil {
	}
	h.hub.SendToClient(c, ws.EventWebRTCAnswer, answer)
}

func (h *wsHandler) handleWebRTCCandidate(c *ws.Client, evt ws.Event) {
	var ice pion.ICECandidateInit
	err := json.Unmarshal(evt.Payload, &ice)
	if err != nil {
	}
	webrtc.AddICECandidate(c.PC, ice)
}

func (h *wsHandler) handleWebRTCAnswer(c *ws.Client, evt ws.Event) {
	var answer pion.SessionDescription
	err := json.Unmarshal(evt.Payload, &answer)
	if err != nil {
	}
	webrtc.HandleAnswer(c.PC, answer)
}
