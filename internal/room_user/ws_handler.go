package roomuser

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"sound-stage-backend/internal/config"
	"sound-stage-backend/internal/pkg/httpx"
	webrtc "sound-stage-backend/internal/web_rtc"
	"sound-stage-backend/internal/ws"

	pion "github.com/pion/webrtc/v4"
)

type setHandRaisedPayload struct {
	IsHandRaised bool `json:"isHandRaised"`
}

type setMutedPayload struct {
	IsMuted bool `json:"isMuted"`
}

type mediaRouter interface {
	Session(clientID string) *webrtc.Session
	AddSession(clientID string, pc *pion.PeerConnection) *webrtc.Session
	FanOutTrack(c *ws.Client, session *webrtc.Session, track *pion.TrackLocalStaticRTP)
	SubscribeToRoomTracks(c *ws.Client, session *webrtc.Session)
	StopPublishing(c *ws.Client)
	CloseSession(clientID string) error
}

type wsAuthorizer interface {
	CanJoinRoom(roomID, userID uint) error
}

type webSocketHub interface {
	BroadcastToRoom(roomID uint, eventName ws.EventName, payload any)
	ErrorToClient(c *ws.Client, message string, statusCode int)
	SendToClient(c *ws.Client, eventName ws.EventName, payload any)
}

type roomUserWSService interface {
	FindBy(userID uint, roomID uint) (*RoomUser, error)
	FindByWithState(ctx context.Context, userID, roomID uint) (*RoomUser, error)
	RemoveUser(ctx context.Context, userID uint, roomID uint) (*RoomUser, error)
	CountsByRoomID(roomID uint) (RoomUserCounts, error)
	SetMuted(ctx context.Context, roomID, userID, actorID uint, isMuted bool) error
	SetHandRaised(ctx context.Context, roomID, userID uint, isHandRaised bool) error
}

type WsHandler struct {
	hub     webSocketHub
	service roomUserWSService
	authz   wsAuthorizer
	media   mediaRouter
	cfg     *config.Config
	logger  *slog.Logger
}

func NewWSHandler(hub webSocketHub, roomUserSvc roomUserWSService, authz wsAuthorizer,
	media mediaRouter, cfg *config.Config, logger *slog.Logger) *WsHandler {
	return &WsHandler{
		hub:     hub,
		service: roomUserSvc,
		authz:   authz,
		media:   media,
		cfg:     cfg,
		logger:  logger,
	}
}

func (h *WsHandler) Register(wsh ws.Handler) {
	wsh.On(ws.EventJoinStream, h.handleUserJoined)
	wsh.On(ws.EventLeaveRoom, h.handleUserLeft)

	wsh.On(ws.EventSetMuted, h.handleSetMuted)
	wsh.On(ws.EventSetHandRaised, h.handleSetHandRaised)

	wsh.On(ws.EventWebRTCOffer, h.handleWebRTCOffer)
	wsh.On(ws.EventWebRTCCandidate, h.handleWebRTCCandidate)
	wsh.On(ws.EventWebRTCAnswer, h.handleWebRTCAnswer)

	wsh.OnDisconnect(h.handleClientDisconnected)
}

func (h *WsHandler) handleUserJoined(c *ws.Client, evt ws.Event) {
	if !h.checkMembership(c) {
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

	session := h.media.AddSession(c.ID, pc)

	h.media.SubscribeToRoomTracks(c, session)

	pc.OnTrack(func(tr *pion.TrackRemote, r *pion.RTPReceiver) {
		ru, err := h.service.FindBy(c.UserID, c.RoomID)
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

		h.media.StopPublishing(c)
		go webrtc.ForwardRTP(tr, localTrack, session.StartPublishing(localTrack))

		h.media.FanOutTrack(c, session, localTrack)
	})

	ru, err := h.service.FindByWithState(context.Background(), c.UserID, c.RoomID)
	if err != nil || ru == nil {
		h.hub.ErrorToClient(c, "Failed to find user in room", http.StatusInternalServerError)
		return
	}

	counts, err := h.service.CountsByRoomID(c.RoomID)
	if err != nil {
		h.hub.ErrorToClient(c, "Failed to fetch room counts", http.StatusInternalServerError)
		return
	}

	h.hub.BroadcastToRoom(c.RoomID, ws.EventJoinRoom, RoomUserEventPayload{
		RoomUserCounts: counts,
		RoomUser:       BuildRoomUserResponse(ru, 0),
	})
}

func (h *WsHandler) handleUserLeft(c *ws.Client, evt ws.Event) {
	ru, err := h.service.RemoveUser(context.Background(), c.UserID, c.RoomID)
	if err != nil {
		h.hub.ErrorToClient(c, "Failed to remove user from room", http.StatusUnprocessableEntity)
		return
	}

	h.media.StopPublishing(c)
	_ = h.media.CloseSession(c.ID)

	h.broadcastUserLeft(c.RoomID, c.UserID, ru.CanSpeak())
}

func (h *WsHandler) handleClientDisconnected(c *ws.Client) {
	ru, err := h.service.RemoveUser(context.Background(), c.UserID, c.RoomID)
	if err != nil {
		h.logger.Error("Failed to remove disconnected user from room",
			slog.Uint64("userId", uint64(c.UserID)),
			slog.Uint64("roomId", uint64(c.RoomID)),
			slog.Any("error", err))
	}

	h.media.StopPublishing(c)

	if err := h.media.CloseSession(c.ID); err != nil {
		h.logger.Error("Failed to close session",
			slog.String("clientId", c.ID), slog.Any("error", err))
	}

	h.broadcastUserLeft(c.RoomID, c.UserID, ru != nil && ru.CanSpeak())
}

func (h *WsHandler) broadcastUserLeft(roomID, userID uint, canSpeak bool) {
	counts, err := h.service.CountsByRoomID(roomID)
	if err != nil {
		h.logger.Error("Failed to fetch room user counts",
			slog.Uint64("roomId", uint64(roomID)),
			slog.Any("error", err))
		return
	}

	h.hub.BroadcastToRoom(roomID, ws.EventLeaveRoom, RoomUserRemovedPayload{
		RoomUserCounts: counts,
		UserID:         userID,
		CanSpeak:       canSpeak,
	})
}

func (h *WsHandler) handleSetMuted(c *ws.Client, evt ws.Event) {
	var p setMutedPayload
	if err := json.Unmarshal(evt.Payload, &p); err != nil {
		h.hub.ErrorToClient(c, "Invalid mute payload", http.StatusUnprocessableEntity)
		return
	}

	if err := h.service.SetMuted(context.Background(), c.RoomID, c.UserID, c.UserID, p.IsMuted); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, httpx.ErrUserBlocked) || errors.Is(err, httpx.ErrForbidden) {
			status = http.StatusForbidden
		}
		h.hub.ErrorToClient(c, "Failed to update mute state", status)
	}
}

func (h *WsHandler) handleSetHandRaised(c *ws.Client, evt ws.Event) {
	var p setHandRaisedPayload
	if err := json.Unmarshal(evt.Payload, &p); err != nil {
		h.hub.ErrorToClient(c, "Invalid hand raised payload", http.StatusUnprocessableEntity)
		return
	}

	if err := h.service.SetHandRaised(context.Background(), c.RoomID, c.UserID, p.IsHandRaised); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, httpx.ErrUserBlocked) {
			status = http.StatusForbidden
		}
		h.hub.ErrorToClient(c, "Failed to update hand raised state", status)
	}
}

func (h *WsHandler) handleWebRTCOffer(c *ws.Client, evt ws.Event) {
	if !h.checkMembership(c) {
		return
	}

	var offer pion.SessionDescription
	if err := json.Unmarshal(evt.Payload, &offer); err != nil {
		h.hub.ErrorToClient(c, "Invalid offer payload", http.StatusUnprocessableEntity)
		return
	}

	session := h.media.Session(c.ID)
	if session == nil || session.PC.SignalingState() != pion.SignalingStateStable {
		return
	}

	answer, err := webrtc.HandleOffer(session.PC, offer)
	if err != nil {
		h.hub.ErrorToClient(c, "Failed to create offer answer", http.StatusUnprocessableEntity)
		return
	}
	if err := session.FlushRemoteCandidates(); err != nil {
		h.hub.ErrorToClient(c, "Failed to add ICE candidate", http.StatusUnprocessableEntity)
	}
	h.hub.SendToClient(c, ws.EventWebRTCAnswer, answer)
}

func (h *WsHandler) handleWebRTCCandidate(c *ws.Client, evt ws.Event) {
	if !h.checkMembership(c) {
		return
	}

	var ice pion.ICECandidateInit
	if err := json.Unmarshal(evt.Payload, &ice); err != nil {
		h.hub.ErrorToClient(c, "Invalid ICE candidate payload", http.StatusUnprocessableEntity)
		return
	}

	session := h.media.Session(c.ID)
	if session == nil {
		return
	}

	if err := session.AddRemoteCandidate(ice); err != nil {
		h.hub.ErrorToClient(c, "Failed to add ICE candidate", http.StatusUnprocessableEntity)
	}
}

func (h *WsHandler) handleWebRTCAnswer(c *ws.Client, evt ws.Event) {
	if !h.checkMembership(c) {
		return
	}

	var answer pion.SessionDescription
	if err := json.Unmarshal(evt.Payload, &answer); err != nil {
		h.hub.ErrorToClient(c, "Invalid answer payload", http.StatusUnprocessableEntity)
		return
	}

	session := h.media.Session(c.ID)
	if session == nil || session.PC.SignalingState() != pion.SignalingStateHaveLocalOffer {
		return
	}

	if err := webrtc.HandleAnswer(session.PC, answer); err != nil {
		h.hub.ErrorToClient(c, "Failed to handle answer", http.StatusUnprocessableEntity)
		return
	}
	if err := session.FlushRemoteCandidates(); err != nil {
		h.hub.ErrorToClient(c, "Failed to add ICE candidate", http.StatusUnprocessableEntity)
	}
}

func (h *WsHandler) checkMembership(c *ws.Client) bool {
	err := h.authz.CanJoinRoom(c.RoomID, c.UserID)
	if err == nil {
		return true
	}
	status := http.StatusInternalServerError
	if errors.Is(err, httpx.ErrForbidden) || errors.Is(err, httpx.ErrUserBlocked) {
		status = http.StatusForbidden
	}
	h.hub.ErrorToClient(c, "You are not a member of this room", status)
	return false
}
