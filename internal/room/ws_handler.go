package room

import (
	"encoding/json"
	"net/http"
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
	cfg             *config.Config
}

func NewWSHandler(hub *ws.Hub, roomUserService roomuser.Service, cfg *config.Config) WSHandler {
	return &wsHandler{
		hub:             hub,
		roomUserService: roomUserService,
		cfg:             cfg,
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
		h.hub.ErrorToClient(c, "Failed to add user to room", http.StatusUnprocessableEntity)
		return
	}

	pc, err := webrtc.NewPeerConnection(
		h.cfg,
		func(ice pion.ICECandidateInit) {
			h.hub.SendToClient(c, ws.EventWebRTCCandidate, ice)
		},
		func(sd pion.SessionDescription) {
			h.hub.SendToClient(c, ws.EventWebRTCOffer, sd)
		})

	if err != nil {
		h.hub.ErrorToClient(c, "Failed to create peer connection", http.StatusInternalServerError)
		return
	}

	c.PC = pc

	pc.OnTrack(func(tr *pion.TrackRemote, r *pion.RTPReceiver) {
		localTrack, err := webrtc.NewForwardingTrack(tr)
		if err != nil {
			h.hub.ErrorToClient(c, "Failed to get track", http.StatusInternalServerError)
			return
		}

		stop := make(chan struct{})
		go webrtc.ForwardRTP(tr, localTrack, stop)

		h.hub.ForEachClientInRoom(c.RoomID, func(client *ws.Client) {
			if client != c {
				_, err := webrtc.AddTrack(client.PC, localTrack)
				if err != nil {
					h.hub.ErrorToClient(client, "Failed to add track", http.StatusInternalServerError)
				}
			}
		})
	})

	h.hub.BroadcastToRoom(c.RoomID, ws.EventJoinRoom, ru)
}

func (h *wsHandler) handleUserLeft(c *ws.Client, evt ws.Event) {
	err := h.roomUserService.RemoveUser(c.UserID, c.RoomID)
	if err != nil {
		h.hub.ErrorToClient(c, "Failed to remove user from room", http.StatusUnprocessableEntity)
		return
	}
	h.hub.BroadcastToRoom(c.RoomID, ws.EventLeaveRoom, nil)
}

func (h *wsHandler) handleWebRTCOffer(c *ws.Client, evt ws.Event) {
	var offer pion.SessionDescription
	err := json.Unmarshal(evt.Payload, &offer)
	answer, err := webrtc.HandleOffer(c.PC, offer)
	if err != nil {
		h.hub.ErrorToClient(c, "Failed to create offer answer", http.StatusUnprocessableEntity)
	}
	h.hub.SendToClient(c, ws.EventWebRTCAnswer, answer)
}

func (h *wsHandler) handleWebRTCCandidate(c *ws.Client, evt ws.Event) {
	var ice pion.ICECandidateInit
	err := json.Unmarshal(evt.Payload, &ice)
	err = webrtc.AddICECandidate(c.PC, ice)
	if err != nil {
		h.hub.ErrorToClient(c, "Failed to add ICE candidate", http.StatusUnprocessableEntity)
	}
}

func (h *wsHandler) handleWebRTCAnswer(c *ws.Client, evt ws.Event) {
	var answer pion.SessionDescription
	err := json.Unmarshal(evt.Payload, &answer)
	err = webrtc.HandleAnswer(c.PC, answer)
	if err != nil {
		h.hub.ErrorToClient(c, "Failed to handle answer", http.StatusUnprocessableEntity)
		return
	}
}
