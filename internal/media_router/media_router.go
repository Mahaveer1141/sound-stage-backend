package mediarouter

import (
	"log/slog"
	"net/http"
	webrtc "sound-stage-backend/internal/web_rtc"
	"sound-stage-backend/internal/ws"

	pion "github.com/pion/webrtc/v4"
)

type wsHub interface {
	ClientsInRoom(roomID uint) []*ws.Client
	ErrorToClient(c *ws.Client, message string, code int)
}

type webrtcSessionStore interface {
	Get(clientID string) *webrtc.Session
	Add(clientID string, pc *pion.PeerConnection) *webrtc.Session
	Remove(clientID string) error
}

type MediaRouter struct {
	hub      wsHub
	sessions webrtcSessionStore
	logger   *slog.Logger
}

func NewMediaRouter(hub wsHub, logger *slog.Logger) *MediaRouter {
	return &MediaRouter{hub: hub, sessions: webrtc.NewSessionStore(), logger: logger}
}

func (m *MediaRouter) Session(clientID string) *webrtc.Session {
	return m.sessions.Get(clientID)
}

func (m *MediaRouter) AddSession(clientID string, pc *pion.PeerConnection) *webrtc.Session {
	return m.sessions.Add(clientID, pc)
}

func (m *MediaRouter) CloseSession(clientID string) error {
	return m.sessions.Remove(clientID)
}

// Add user as senders (from whom they are receiving tracks) to all publishing clients in the room
func (m *MediaRouter) SubscribeToRoomTracks(c *ws.Client, session *webrtc.Session) {
	for _, client := range m.hub.ClientsInRoom(c.RoomID) {
		if client.ID == c.ID {
			continue
		}

		peer := m.sessions.Get(client.ID)
		if peer == nil || peer.LocalTrack() == nil {
			continue
		}

		sender, err := webrtc.AddTrack(session.PC, peer.LocalTrack())
		if err != nil {
			m.hub.ErrorToClient(c, "Failed to add track", http.StatusInternalServerError)
			continue
		}
		peer.AddSender(c.ID, sender)
	}
}

// Add all active clients in the room in user's senders(to whom we are sending tracks)
func (m *MediaRouter) FanOutTrack(c *ws.Client, session *webrtc.Session, track *pion.TrackLocalStaticRTP) {
	for _, client := range m.hub.ClientsInRoom(c.RoomID) {
		if client.ID == c.ID {
			continue
		}

		peer := m.sessions.Get(client.ID)
		if peer == nil {
			continue
		}

		sender, err := webrtc.AddTrack(peer.PC, track)
		if err != nil {
			m.hub.ErrorToClient(client, "Failed to add track", http.StatusInternalServerError)
			continue
		}
		session.AddSender(client.ID, sender)
	}
}

func (m *MediaRouter) RevokePublishing(roomID uint, userID uint) {
	for _, c := range m.hub.ClientsInRoom(roomID) {
		if c.UserID == userID {
			m.StopPublishing(c)
		}
	}
}

func (m *MediaRouter) StopPublishing(c *ws.Client) {
	session := m.sessions.Get(c.ID)
	if session == nil {
		return
	}

	for clientID, sender := range session.StopPublishing() {
		subscriber := m.sessions.Get(clientID)
		if subscriber == nil {
			continue
		}

		if err := webrtc.RemoveTrack(subscriber.PC, sender); err != nil {
			m.logger.Error("Failed to detach track from subscriber",
				slog.String("subscriberId", clientID),
				slog.String("publisherId", c.ID),
				slog.Any("error", err))
		}
	}
}
