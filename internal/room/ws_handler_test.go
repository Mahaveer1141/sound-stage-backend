package room

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"testing"

	pion "github.com/pion/webrtc/v4"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"sound-stage-backend/internal/config"
	"sound-stage-backend/internal/role"
	roomuser "sound-stage-backend/internal/room_user"
	webrtc "sound-stage-backend/internal/web_rtc"
	"sound-stage-backend/internal/ws"
)

type mockMediaRouter struct{ mock.Mock }

func (m *mockMediaRouter) Session(clientID string) *webrtc.Session {
	args := m.Called(clientID)
	s, _ := args.Get(0).(*webrtc.Session)
	return s
}
func (m *mockMediaRouter) AddSession(clientID string, pc *pion.PeerConnection) *webrtc.Session {
	args := m.Called(clientID, pc)
	s, _ := args.Get(0).(*webrtc.Session)
	return s
}
func (m *mockMediaRouter) FanOutTrack(c *ws.Client, session *webrtc.Session, track *pion.TrackLocalStaticRTP) {
	m.Called(c, session, track)
}
func (m *mockMediaRouter) SubscribeToRoomTracks(c *ws.Client, session *webrtc.Session) {
	m.Called(c, session)
}
func (m *mockMediaRouter) StopPublishing(c *ws.Client) {
	m.Called(c)
}
func (m *mockMediaRouter) CloseSession(clientID string) error {
	args := m.Called(clientID)
	return args.Error(0)
}

type mockWSHub struct{ mock.Mock }

func (m *mockWSHub) BroadcastToRoom(roomID uint, eventName ws.EventName, payload any) {
	m.Called(roomID, eventName, payload)
}
func (m *mockWSHub) ErrorToClient(c *ws.Client, message string, statusCode int) {
	m.Called(c, message, statusCode)
}
func (m *mockWSHub) SendToClient(c *ws.Client, eventName ws.EventName, payload any) {
	m.Called(c, eventName, payload)
}

type wsHarness struct {
	roomUser  *mockRoomUserService
	media     *mockMediaRouter
	hub       *mockWSHub
	wsHandler *WsHandler
}

func newWSHarness(t *testing.T) *wsHarness {
	t.Helper()
	ru := new(mockRoomUserService)
	media := new(mockMediaRouter)
	hub := new(mockWSHub)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.Config{WebRTC: config.WebRTCConfig{StunURL: "stun:stun.l.google.com:19302"}}
	return &wsHarness{
		roomUser:  ru,
		media:     media,
		hub:       hub,
		wsHandler: NewWSHandler(hub, ru, media, cfg, logger),
	}
}

func testClient() *ws.Client {
	return &ws.Client{ID: "client-1", UserID: 42, RoomID: 4}
}

type simpleError struct{ msg string }

func (e *simpleError) Error() string { return e.msg }

func errLike(msg string) error { return &simpleError{msg} }

func newPeerConnectionInState(t *testing.T, state pion.SignalingState) *pion.PeerConnection {
	t.Helper()
	pc, err := pion.NewPeerConnection(pion.Configuration{})
	require.NoError(t, err)
	t.Cleanup(func() { _ = pc.Close() })

	switch state {
	case pion.SignalingStateStable:
		// freshly created PeerConnection starts stable; nothing further needed.
	case pion.SignalingStateHaveLocalOffer:
		offer, err := pc.CreateOffer(nil)
		require.NoError(t, err)
		require.NoError(t, pc.SetLocalDescription(offer))
	case pion.SignalingStateHaveRemoteOffer:
		_, err := pc.AddTransceiverFromKind(pion.RTPCodecTypeAudio)
		require.NoError(t, err)
		offer, err := pc.CreateOffer(nil)
		require.NoError(t, err)
		require.NoError(t, pc.SetRemoteDescription(offer))
	default:
		t.Fatalf("unsupported state for test helper: %v", state)
	}
	return pc
}

func TestWsHandler_handleUserJoined(t *testing.T) {
	t.Run("success: adds user, creates session, subscribes, broadcasts join", func(t *testing.T) {
		h := newWSHarness(t)
		c := testClient()
		ru := &roomuser.RoomUser{UserID: 42, RoomID: 4}

		h.roomUser.On("AddUser", uint(42), uint(4), role.RoleListener).Return(ru, nil)
		h.media.On("AddSession", "client-1", mock.AnythingOfType("*webrtc.PeerConnection")).
			Return(&webrtc.Session{})
		h.media.On("SubscribeToRoomTracks", c, mock.Anything).Return()
		h.hub.On("BroadcastToRoom", uint(4), ws.EventJoinRoom, ru).Return()

		h.wsHandler.handleUserJoined(c, ws.Event{})

		h.roomUser.AssertExpectations(t)
		h.media.AssertExpectations(t)
		h.hub.AssertExpectations(t)
	})

	t.Run("failure: AddUser error sends error to client, never creates a session", func(t *testing.T) {
		h := newWSHarness(t)
		c := testClient()
		h.roomUser.On("AddUser", uint(42), uint(4), role.RoleListener).Return(nil, errLike("failed to add user"))
		h.hub.On("ErrorToClient", c, "Failed to add user to room", http.StatusUnprocessableEntity).Return()

		h.wsHandler.handleUserJoined(c, ws.Event{})

		h.media.AssertNotCalled(t, "AddSession", mock.Anything, mock.Anything)
		h.roomUser.AssertExpectations(t)
		h.hub.AssertExpectations(t)
	})
}

func TestWsHandler_handleUserLeft(t *testing.T) {
	t.Run("success: removes user, stops publishing, closes session, broadcasts leave", func(t *testing.T) {
		h := newWSHarness(t)
		c := testClient()
		h.roomUser.On("RemoveUser", uint(42), uint(4)).Return(nil)
		h.media.On("StopPublishing", c).Return()
		h.media.On("CloseSession", "client-1").Return(nil)
		h.hub.On("BroadcastToRoom", uint(4), ws.EventLeaveRoom, nil).Return()

		h.wsHandler.handleUserLeft(c, ws.Event{})

		h.roomUser.AssertExpectations(t)
		h.media.AssertExpectations(t)
		h.hub.AssertExpectations(t)
	})

	t.Run("failure: RemoveUser error sends error to client, no cleanup or broadcast", func(t *testing.T) {
		h := newWSHarness(t)
		c := testClient()
		h.roomUser.On("RemoveUser", uint(42), uint(4)).Return(errLike("failed to remove user"))
		h.hub.On("ErrorToClient", c, "Failed to remove user from room", http.StatusUnprocessableEntity).Return()

		h.wsHandler.handleUserLeft(c, ws.Event{})

		h.media.AssertNotCalled(t, "StopPublishing", mock.Anything)
		h.media.AssertNotCalled(t, "CloseSession", mock.Anything)
		h.hub.AssertNotCalled(t, "BroadcastToRoom", mock.Anything, mock.Anything, mock.Anything)
		h.roomUser.AssertExpectations(t)
		h.hub.AssertExpectations(t)
	})
}

func TestWsHandler_handleClientDisconnected(t *testing.T) {
	t.Run("success: cleans up and broadcasts leave with no errors", func(t *testing.T) {
		h := newWSHarness(t)
		c := testClient()
		h.roomUser.On("RemoveUser", uint(42), uint(4)).Return(nil)
		h.media.On("StopPublishing", c).Return()
		h.media.On("CloseSession", "client-1").Return(nil)
		h.hub.On("BroadcastToRoom", uint(4), ws.EventLeaveRoom, nil).Return()

		h.wsHandler.handleClientDisconnected(c)

		h.roomUser.AssertExpectations(t)
		h.media.AssertExpectations(t)
		h.hub.AssertExpectations(t)
	})

	t.Run("failure: RemoveUser and CloseSession errors are only logged, cleanup and broadcast still run", func(t *testing.T) {
		h := newWSHarness(t)
		c := testClient()
		h.roomUser.On("RemoveUser", uint(42), uint(4)).Return(errLike("db down"))
		h.media.On("StopPublishing", c).Return()
		h.media.On("CloseSession", "client-1").Return(errLike("close failed"))
		h.hub.On("BroadcastToRoom", uint(4), ws.EventLeaveRoom, nil).Return()

		h.wsHandler.handleClientDisconnected(c)

		h.roomUser.AssertExpectations(t)
		h.media.AssertExpectations(t)
		h.hub.AssertExpectations(t)
		h.hub.AssertNotCalled(t, "ErrorToClient", mock.Anything, mock.Anything, mock.Anything)
	})
}

func TestWsHandler_handleWebRTCOffer(t *testing.T) {
	t.Run("failure: malformed payload sends error to client, no session lookup", func(t *testing.T) {
		h := newWSHarness(t)
		c := testClient()
		h.hub.On("ErrorToClient", c, "Invalid offer payload", http.StatusUnprocessableEntity).Return()

		h.wsHandler.handleWebRTCOffer(c, ws.Event{Payload: json.RawMessage(`{not json`)})

		h.media.AssertNotCalled(t, "Session", mock.Anything)
		h.hub.AssertExpectations(t)
	})

	t.Run("failure: no session for client is a silent no-op", func(t *testing.T) {
		h := newWSHarness(t)
		c := testClient()
		offer := pion.SessionDescription{Type: pion.SDPTypeOffer, SDP: "v=0"}
		payload, _ := json.Marshal(offer)
		h.media.On("Session", "client-1").Return(nil)

		h.wsHandler.handleWebRTCOffer(c, ws.Event{Payload: payload})

		h.hub.AssertNotCalled(t, "ErrorToClient", mock.Anything, mock.Anything, mock.Anything)
		h.hub.AssertNotCalled(t, "SendToClient", mock.Anything, mock.Anything, mock.Anything)
		h.media.AssertExpectations(t)
	})

	t.Run("failure: session not in stable state is a silent no-op (glare protection)", func(t *testing.T) {
		h := newWSHarness(t)
		c := testClient()
		offer := pion.SessionDescription{Type: pion.SDPTypeOffer, SDP: "v=0"}
		payload, _ := json.Marshal(offer)
		pc := newPeerConnectionInState(t, pion.SignalingStateHaveLocalOffer)
		h.media.On("Session", "client-1").Return(&webrtc.Session{PC: pc})

		h.wsHandler.handleWebRTCOffer(c, ws.Event{Payload: payload})

		h.hub.AssertNotCalled(t, "ErrorToClient", mock.Anything, mock.Anything, mock.Anything)
		h.hub.AssertNotCalled(t, "SendToClient", mock.Anything, mock.Anything, mock.Anything)
		h.media.AssertExpectations(t)
	})

	t.Run("stable session with garbage offer: HandleOffer error surfaces to client", func(t *testing.T) {
		h := newWSHarness(t)
		c := testClient()
		offer := pion.SessionDescription{Type: pion.SDPTypeOffer, SDP: "v=0"}
		payload, _ := json.Marshal(offer)
		pc := newPeerConnectionInState(t, pion.SignalingStateStable)
		h.media.On("Session", "client-1").Return(&webrtc.Session{PC: pc})
		h.hub.On("ErrorToClient", c, "Failed to create offer answer", http.StatusUnprocessableEntity).Return()

		h.wsHandler.handleWebRTCOffer(c, ws.Event{Payload: payload})

		h.media.AssertExpectations(t)
		h.hub.AssertExpectations(t)
	})
}

func TestWsHandler_handleWebRTCCandidate(t *testing.T) {
	t.Run("success: valid candidate on active session is added without error", func(t *testing.T) {
		h := newWSHarness(t)
		c := testClient()
		ice := pion.ICECandidateInit{Candidate: "candidate:1 1 UDP 1 127.0.0.1 9 typ host"}
		payload, _ := json.Marshal(ice)
		pc := newPeerConnectionInState(t, pion.SignalingStateHaveRemoteOffer)
		h.media.On("Session", "client-1").Return(&webrtc.Session{PC: pc})

		h.wsHandler.handleWebRTCCandidate(c, ws.Event{Payload: payload})

		h.media.AssertExpectations(t)
		h.hub.AssertNotCalled(t, "ErrorToClient", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("failure: malformed payload sends error to client, no session lookup", func(t *testing.T) {
		h := newWSHarness(t)
		c := testClient()
		h.hub.On("ErrorToClient", c, "Invalid ICE candidate payload", http.StatusUnprocessableEntity).Return()

		h.wsHandler.handleWebRTCCandidate(c, ws.Event{Payload: json.RawMessage(`{not json`)})

		h.media.AssertNotCalled(t, "Session", mock.Anything)
		h.hub.AssertExpectations(t)
	})

	t.Run("failure: no session for client is a silent no-op", func(t *testing.T) {
		h := newWSHarness(t)
		c := testClient()
		ice := pion.ICECandidateInit{Candidate: "candidate:1 1 UDP 1 127.0.0.1 9 typ host"}
		payload, _ := json.Marshal(ice)
		h.media.On("Session", "client-1").Return(nil)

		h.wsHandler.handleWebRTCCandidate(c, ws.Event{Payload: payload})

		h.hub.AssertNotCalled(t, "ErrorToClient", mock.Anything, mock.Anything, mock.Anything)
		h.media.AssertExpectations(t)
	})
}

func TestWsHandler_handleWebRTCAnswer(t *testing.T) {
	t.Run("failure: malformed payload sends error to client, no session lookup", func(t *testing.T) {
		h := newWSHarness(t)
		c := testClient()
		h.hub.On("ErrorToClient", c, "Invalid answer payload", http.StatusUnprocessableEntity).Return()

		h.wsHandler.handleWebRTCAnswer(c, ws.Event{Payload: json.RawMessage(`{not json`)})

		h.media.AssertNotCalled(t, "Session", mock.Anything)
		h.hub.AssertExpectations(t)
	})

	t.Run("failure: no session for client is a silent no-op", func(t *testing.T) {
		h := newWSHarness(t)
		c := testClient()
		answer := pion.SessionDescription{Type: pion.SDPTypeAnswer, SDP: "v=0"}
		payload, _ := json.Marshal(answer)
		h.media.On("Session", "client-1").Return(nil)

		h.wsHandler.handleWebRTCAnswer(c, ws.Event{Payload: payload})

		h.hub.AssertNotCalled(t, "ErrorToClient", mock.Anything, mock.Anything, mock.Anything)
		h.media.AssertExpectations(t)
	})

	t.Run("failure: session not awaiting a local offer is a silent no-op", func(t *testing.T) {
		h := newWSHarness(t)
		c := testClient()
		answer := pion.SessionDescription{Type: pion.SDPTypeAnswer, SDP: "v=0"}
		payload, _ := json.Marshal(answer)
		pc := newPeerConnectionInState(t, pion.SignalingStateStable)
		h.media.On("Session", "client-1").Return(&webrtc.Session{PC: pc})

		h.wsHandler.handleWebRTCAnswer(c, ws.Event{Payload: payload})

		h.hub.AssertNotCalled(t, "ErrorToClient", mock.Anything, mock.Anything, mock.Anything)
		h.media.AssertExpectations(t)
	})

	t.Run("awaiting-offer session with garbage answer: HandleAnswer error surfaces to client", func(t *testing.T) {
		h := newWSHarness(t)
		c := testClient()
		answer := pion.SessionDescription{Type: pion.SDPTypeAnswer, SDP: "v=0"}
		payload, _ := json.Marshal(answer)
		pc := newPeerConnectionInState(t, pion.SignalingStateHaveLocalOffer)
		h.media.On("Session", "client-1").Return(&webrtc.Session{PC: pc})
		h.hub.On("ErrorToClient", c, "Failed to handle answer", http.StatusUnprocessableEntity).Return()

		h.wsHandler.handleWebRTCAnswer(c, ws.Event{Payload: payload})

		h.media.AssertExpectations(t)
		h.hub.AssertExpectations(t)
	})
}
