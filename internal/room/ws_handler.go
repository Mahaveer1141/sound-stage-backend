package room

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sound-stage-backend/internal/config"
	"sound-stage-backend/internal/role"
	roomuser "sound-stage-backend/internal/room_user"
	webrtc "sound-stage-backend/internal/web_rtc"
	"sound-stage-backend/internal/ws"

	pion "github.com/pion/webrtc/v4"
)

type WSHandler interface {
	Register(wsh ws.Handler)
	handleUserJoined(c *ws.Client, evt ws.Event)
	handleUserLeft(c *ws.Client, evt ws.Event)
	handleClientDisconnected(c *ws.Client)
	handleWebRTCOffer(c *ws.Client, evt ws.Event)
	handleWebRTCCandidate(c *ws.Client, evt ws.Event)
	handleWebRTCAnswer(c *ws.Client, evt ws.Event)
}

type wsHandler struct {
	hub             *ws.Hub
	roomUserService roomuser.Service
	sessions        *webrtc.SessionStore
	cfg             *config.Config
	logger          *slog.Logger
}

func NewWSHandler(hub *ws.Hub, roomUserService roomuser.Service, cfg *config.Config, logger *slog.Logger) WSHandler {
	return &wsHandler{
		hub:             hub,
		roomUserService: roomUserService,
		sessions:        webrtc.NewSessionStore(),
		cfg:             cfg,
		logger:          logger,
	}
}

func (h *wsHandler) Register(wsh ws.Handler) {
	wsh.On(ws.EventJoinRoom, h.handleUserJoined)
	wsh.On(ws.EventLeaveRoom, h.handleUserLeft)

	wsh.On(ws.EventWebRTCOffer, h.handleWebRTCOffer)
	wsh.On(ws.EventWebRTCCandidate, h.handleWebRTCCandidate)
	wsh.On(ws.EventWebRTCAnswer, h.handleWebRTCAnswer)

	wsh.OnDisconnect(h.handleClientDisconnected)
}

func (h *wsHandler) handleUserJoined(c *ws.Client, evt ws.Event) {
	ru, err := h.roomUserService.AddUser(c.UserID, c.RoomID, role.RoleListener)
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

	session := h.sessions.Add(c.ID, pc)

	h.subscribeToRoomTracks(c, session)

	pc.OnTrack(func(tr *pion.TrackRemote, r *pion.RTPReceiver) {
		ru, err := h.roomUserService.FindBy(c.UserID, c.RoomID)
		if err != nil {
			h.hub.ErrorToClient(c, "Failed to find user in room", http.StatusInternalServerError)
			return
		}
		if ru.IsListener() {
			return
		}

		localTrack, err := webrtc.NewForwardingTrack(tr)
		if err != nil {
			h.hub.ErrorToClient(c, "Failed to get track", http.StatusInternalServerError)
			return
		}
		session.SetLocalTrack(localTrack)

		go webrtc.ForwardRTP(tr, localTrack, session.Stop())

		h.fanOutTrack(c, localTrack)
	})

	h.hub.BroadcastToRoom(c.RoomID, ws.EventJoinRoom, ru)
}

func (h *wsHandler) subscribeToRoomTracks(c *ws.Client, session *webrtc.Session) {
	for _, client := range h.hub.ClientsInRoom(c.RoomID) {
		if client.ID == c.ID {
			continue
		}

		peer := h.sessions.Get(client.ID)
		if peer == nil || peer.LocalTrack() == nil {
			continue
		}

		if _, err := webrtc.AddTrack(session.PC, peer.LocalTrack()); err != nil {
			h.hub.ErrorToClient(c, "Failed to add track", http.StatusInternalServerError)
		}
	}
}

func (h *wsHandler) fanOutTrack(c *ws.Client, track *pion.TrackLocalStaticRTP) {
	for _, client := range h.hub.ClientsInRoom(c.RoomID) {
		if client.ID == c.ID {
			continue
		}

		peer := h.sessions.Get(client.ID)
		if peer == nil {
			continue
		}

		if _, err := webrtc.AddTrack(peer.PC, track); err != nil {
			h.hub.ErrorToClient(client, "Failed to add track", http.StatusInternalServerError)
		}
	}
}

func (h *wsHandler) handleUserLeft(c *ws.Client, evt ws.Event) {
	err := h.roomUserService.RemoveUser(c.UserID, c.RoomID)
	if err != nil {
		h.hub.ErrorToClient(c, "Failed to remove user from room", http.StatusUnprocessableEntity)
		return
	}

	_ = h.sessions.Remove(c.ID)
	h.hub.BroadcastToRoom(c.RoomID, ws.EventLeaveRoom, nil)
}

func (h *wsHandler) handleClientDisconnected(c *ws.Client) {
	if err := h.roomUserService.RemoveUser(c.UserID, c.RoomID); err != nil {
		h.logger.Error("Failed to remove disconnected user from room",
			slog.Uint64("userId", uint64(c.UserID)),
			slog.Uint64("roomId", uint64(c.RoomID)),
			slog.Any("error", err))
	}

	if err := h.sessions.Remove(c.ID); err != nil {
		h.logger.Error("Failed to close session",
			slog.String("clientId", c.ID), slog.Any("error", err))
	}

	h.hub.BroadcastToRoom(c.RoomID, ws.EventLeaveRoom, nil)
}

func (h *wsHandler) handleWebRTCOffer(c *ws.Client, evt ws.Event) {
	var offer pion.SessionDescription
	if err := json.Unmarshal(evt.Payload, &offer); err != nil {
		h.hub.ErrorToClient(c, "Invalid offer payload", http.StatusUnprocessableEntity)
		return
	}

	session := h.sessions.Get(c.ID)
	if session == nil || session.PC.SignalingState() != pion.SignalingStateStable {
		return
	}

	answer, err := webrtc.HandleOffer(session.PC, offer)
	if err != nil {
		h.hub.ErrorToClient(c, "Failed to create offer answer", http.StatusUnprocessableEntity)
		return
	}
	h.hub.SendToClient(c, ws.EventWebRTCAnswer, answer)
}

func (h *wsHandler) handleWebRTCCandidate(c *ws.Client, evt ws.Event) {
	var ice pion.ICECandidateInit
	if err := json.Unmarshal(evt.Payload, &ice); err != nil {
		h.hub.ErrorToClient(c, "Invalid ICE candidate payload", http.StatusUnprocessableEntity)
		return
	}

	session := h.sessions.Get(c.ID)
	if session == nil {
		return
	}

	if err := webrtc.AddICECandidate(session.PC, ice); err != nil {
		h.hub.ErrorToClient(c, "Failed to add ICE candidate", http.StatusUnprocessableEntity)
	}
}

func (h *wsHandler) handleWebRTCAnswer(c *ws.Client, evt ws.Event) {
	var answer pion.SessionDescription
	if err := json.Unmarshal(evt.Payload, &answer); err != nil {
		h.hub.ErrorToClient(c, "Invalid answer payload", http.StatusUnprocessableEntity)
		return
	}

	session := h.sessions.Get(c.ID)
	if session == nil || session.PC.SignalingState() != pion.SignalingStateHaveLocalOffer {
		return
	}

	if err := webrtc.HandleAnswer(session.PC, answer); err != nil {
		h.hub.ErrorToClient(c, "Failed to handle answer", http.StatusUnprocessableEntity)
	}
}
