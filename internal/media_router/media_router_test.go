package mediarouter

import (
	"io"
	"log/slog"
	"net/http"
	"testing"

	pion "github.com/pion/webrtc/v4"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	webrtc "sound-stage-backend/internal/web_rtc"
	"sound-stage-backend/internal/ws"
)

type mockWSHub struct{ mock.Mock }

func (m *mockWSHub) ClientsInRoom(roomID uint) []*ws.Client {
	args := m.Called(roomID)
	clients, _ := args.Get(0).([]*ws.Client)
	return clients
}
func (m *mockWSHub) ErrorToClient(c *ws.Client, message string, code int) {
	m.Called(c, message, code)
}

type mockSessionStore struct{ mock.Mock }

func (m *mockSessionStore) Get(clientID string) *webrtc.Session {
	args := m.Called(clientID)
	s, _ := args.Get(0).(*webrtc.Session)
	return s
}
func (m *mockSessionStore) Add(clientID string, pc *pion.PeerConnection) *webrtc.Session {
	args := m.Called(clientID, pc)
	s, _ := args.Get(0).(*webrtc.Session)
	return s
}
func (m *mockSessionStore) Remove(clientID string) error {
	args := m.Called(clientID)
	return args.Error(0)
}

type harness struct {
	hub      *mockWSHub
	sessions *mockSessionStore
	m        *MediaRouter
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	hub := new(mockWSHub)
	sessions := new(mockSessionStore)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return &harness{
		hub:      hub,
		sessions: sessions,
		m: &MediaRouter{
			hub:      hub,
			sessions: sessions,
			logger:   logger,
		},
	}
}

func testClient(id string, userID, roomID uint) *ws.Client {
	return &ws.Client{ID: id, UserID: userID, RoomID: roomID}
}

func newPeerConnection(t *testing.T) *pion.PeerConnection {
	t.Helper()
	pc, err := pion.NewPeerConnection(pion.Configuration{})
	require.NoError(t, err)
	t.Cleanup(func() { _ = pc.Close() })
	return pc
}

func newLocalTrack(t *testing.T, id string) *pion.TrackLocalStaticRTP {
	t.Helper()
	track, err := pion.NewTrackLocalStaticRTP(pion.RTPCodecCapability{MimeType: pion.MimeTypeOpus}, id, "stream")
	require.NoError(t, err)
	return track
}

func newSession(t *testing.T) *webrtc.Session {
	t.Helper()
	return webrtc.NewSessionStore().Add("client", newPeerConnection(t))
}

func TestMediaRouter_Session(t *testing.T) {
	t.Run("success: returns session from store", func(t *testing.T) {
		h := newHarness(t)
		want := newSession(t)
		h.sessions.On("Get", "client-1").Return(want)

		got := h.m.Session("client-1")

		require.Same(t, want, got)
		h.sessions.AssertExpectations(t)
	})

	t.Run("failure: unknown client returns nil, not an error", func(t *testing.T) {
		h := newHarness(t)
		h.sessions.On("Get", "unknown").Return(nil)

		got := h.m.Session("unknown")

		require.Nil(t, got)
		h.sessions.AssertExpectations(t)
	})
}

func TestMediaRouter_AddSession(t *testing.T) {
	t.Run("success: delegates to store and returns the created session", func(t *testing.T) {
		h := newHarness(t)
		pc := newPeerConnection(t)
		want := &webrtc.Session{PC: pc}
		h.sessions.On("Add", "client-1", pc).Return(want)

		got := h.m.AddSession("client-1", pc)

		require.Same(t, want, got)
		h.sessions.AssertExpectations(t)
	})
}

func TestMediaRouter_CloseSession(t *testing.T) {
	t.Run("success: delegates to store", func(t *testing.T) {
		h := newHarness(t)
		h.sessions.On("Remove", "client-1").Return(nil)

		err := h.m.CloseSession("client-1")

		require.NoError(t, err)
		h.sessions.AssertExpectations(t)
	})

	t.Run("failure: store error is propagated", func(t *testing.T) {
		h := newHarness(t)
		removeErr := assertErr("session not found")
		h.sessions.On("Remove", "client-1").Return(removeErr)

		err := h.m.CloseSession("client-1")

		require.ErrorIs(t, err, removeErr)
		h.sessions.AssertExpectations(t)
	})
}

func TestMediaRouter_SubscribeToRoomTracks(t *testing.T) {
	t.Run("success: attaches existing publishers' tracks as senders on the new session", func(t *testing.T) {
		h := newHarness(t)
		self := testClient("client-new", 1, 4)
		newSess := newSession(t)

		publisherPC := newPeerConnection(t)
		publisherTrack := newLocalTrack(t, "audio-1")
		_, err := publisherPC.AddTrack(publisherTrack)
		require.NoError(t, err)
		publisherSession := webrtc.NewSessionStore().Add("client-pub", publisherPC)
		_ = publisherSession.StartPublishing(publisherTrack)
		publisherClient := testClient("client-pub", 2, 4)

		otherClient := testClient("client-idle", 3, 4)

		h.hub.On("ClientsInRoom", uint(4)).Return([]*ws.Client{self, publisherClient, otherClient})
		h.sessions.On("Get", "client-pub").Return(publisherSession)
		h.sessions.On("Get", "client-idle").Return(nil)

		h.m.SubscribeToRoomTracks(self, newSess)

		h.hub.AssertExpectations(t)
		h.sessions.AssertExpectations(t)
		h.hub.AssertNotCalled(t, "ErrorToClient", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("failure: AddTrack error reports to client and continues to remaining peers", func(t *testing.T) {
		h := newHarness(t)
		self := testClient("client-new", 1, 4)
		closedSess := &webrtc.Session{PC: newPeerConnection(t)}
		require.NoError(t, closedSess.PC.Close())

		publisherPC := newPeerConnection(t)
		publisherTrack := newLocalTrack(t, "audio-1")
		_, err := publisherPC.AddTrack(publisherTrack)
		require.NoError(t, err)
		publisherSession := webrtc.NewSessionStore().Add("client-pub", publisherPC)
		_ = publisherSession.StartPublishing(publisherTrack)
		publisherClient := testClient("client-pub", 2, 4)

		h.hub.On("ClientsInRoom", uint(4)).Return([]*ws.Client{self, publisherClient})
		h.sessions.On("Get", "client-pub").Return(publisherSession)
		h.hub.On("ErrorToClient", self, "Failed to add track", http.StatusInternalServerError).Return()

		h.m.SubscribeToRoomTracks(self, closedSess)

		h.hub.AssertExpectations(t)
		h.sessions.AssertExpectations(t)
	})

	t.Run("edge case: self is excluded from the loop even if listed by the hub", func(t *testing.T) {
		h := newHarness(t)
		self := testClient("client-new", 1, 4)
		newSess := newSession(t)

		h.hub.On("ClientsInRoom", uint(4)).Return([]*ws.Client{self})

		h.m.SubscribeToRoomTracks(self, newSess)

		h.hub.AssertExpectations(t)
		h.sessions.AssertNotCalled(t, "Get", mock.Anything)
	})
}

func TestMediaRouter_FanOutTrack(t *testing.T) {
	t.Run("success: adds the new track to every other active client and registers senders", func(t *testing.T) {
		h := newHarness(t)
		self := testClient("client-pub", 1, 4)
		selfSession := newSession(t)
		track := newLocalTrack(t, "audio-new")

		subscriberSession := newSession(t)
		subscriberClient := testClient("client-sub", 2, 4)

		h.hub.On("ClientsInRoom", uint(4)).Return([]*ws.Client{self, subscriberClient})
		h.sessions.On("Get", "client-sub").Return(subscriberSession)

		h.m.FanOutTrack(self, selfSession, track)

		h.hub.AssertExpectations(t)
		h.sessions.AssertExpectations(t)
		h.hub.AssertNotCalled(t, "ErrorToClient", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("failure: AddTrack error on one subscriber reports to that subscriber, others still processed", func(t *testing.T) {
		h := newHarness(t)
		self := testClient("client-pub", 1, 4)
		selfSession := newSession(t)
		track := newLocalTrack(t, "audio-new")

		closedSubSession := &webrtc.Session{PC: newPeerConnection(t)}
		require.NoError(t, closedSubSession.PC.Close())
		closedSubClient := testClient("client-closed", 2, 4)

		healthySubSession := newSession(t)
		healthySubClient := testClient("client-healthy", 3, 4)

		h.hub.On("ClientsInRoom", uint(4)).Return([]*ws.Client{self, closedSubClient, healthySubClient})
		h.sessions.On("Get", "client-closed").Return(closedSubSession)
		h.sessions.On("Get", "client-healthy").Return(healthySubSession)
		h.hub.On("ErrorToClient", closedSubClient, "Failed to add track", http.StatusInternalServerError).Return()

		h.m.FanOutTrack(self, selfSession, track)

		h.hub.AssertExpectations(t)
		h.sessions.AssertExpectations(t)
	})

	t.Run("edge case: unknown session for a listed client is skipped", func(t *testing.T) {
		h := newHarness(t)
		self := testClient("client-pub", 1, 4)
		selfSession := newSession(t)
		track := newLocalTrack(t, "audio-new")
		unknownClient := testClient("client-gone", 2, 4)

		h.hub.On("ClientsInRoom", uint(4)).Return([]*ws.Client{self, unknownClient})
		h.sessions.On("Get", "client-gone").Return(nil)

		h.m.FanOutTrack(self, selfSession, track)

		h.hub.AssertExpectations(t)
		h.hub.AssertNotCalled(t, "ErrorToClient", mock.Anything, mock.Anything, mock.Anything)
	})
}

func TestMediaRouter_RevokePublishing(t *testing.T) {
	t.Run("success: stops publishing only for the matching user's client", func(t *testing.T) {
		h := newHarness(t)
		target := testClient("client-target", 5, 4)
		other := testClient("client-other", 6, 4)
		targetSession := newSession(t)

		h.hub.On("ClientsInRoom", uint(4)).Return([]*ws.Client{target, other})
		h.sessions.On("Get", "client-target").Return(targetSession)

		h.m.RevokePublishing(4, 5)

		h.hub.AssertExpectations(t)
		h.sessions.AssertNotCalled(t, "Get", "client-other")
	})

	t.Run("edge case: user not present in room is a no-op", func(t *testing.T) {
		h := newHarness(t)
		other := testClient("client-other", 6, 4)

		h.hub.On("ClientsInRoom", uint(4)).Return([]*ws.Client{other})

		h.m.RevokePublishing(4, 999)

		h.hub.AssertExpectations(t)
		h.sessions.AssertNotCalled(t, "Get", mock.Anything)
	})
}

func TestMediaRouter_StopPublishing(t *testing.T) {
	t.Run("success: no session is a silent no-op", func(t *testing.T) {
		h := newHarness(t)
		c := testClient("client-1", 1, 4)
		h.sessions.On("Get", "client-1").Return(nil)

		h.m.StopPublishing(c)

		h.sessions.AssertExpectations(t)
	})

	t.Run("success: session with no active subscribers still calls Get once and completes", func(t *testing.T) {
		h := newHarness(t)
		c := testClient("client-1", 1, 4)
		sess := newSession(t)
		h.sessions.On("Get", "client-1").Return(sess)

		h.m.StopPublishing(c)

		h.sessions.AssertExpectations(t)
	})
}

type simpleError struct{ msg string }

func (e *simpleError) Error() string { return e.msg }
func assertErr(msg string) error     { return &simpleError{msg} }
