package chatmessage

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"sound-stage-backend/internal/model"
	"sound-stage-backend/internal/pkg/httpx"
	"sound-stage-backend/internal/ws"
)

type mockChatMessageServiceWS struct{ mock.Mock }

func (m *mockChatMessageServiceWS) Create(input *CreateChatMessageParams) (*ChatMessage, error) {
	args := m.Called(input)
	msg, _ := args.Get(0).(*ChatMessage)
	return msg, args.Error(1)
}

type mockWSHub struct{ mock.Mock }

func (m *mockWSHub) BroadcastToRoom(roomID uint, eventName ws.EventName, payload any) {
	m.Called(roomID, eventName, payload)
}

func (m *mockWSHub) ErrorToClient(c *ws.Client, message string, statusCode int) {
	m.Called(c, message, statusCode)
}

type wsHarness struct {
	svc       *mockChatMessageServiceWS
	hub       *mockWSHub
	wsHandler *WsHandler
}

func newWSHarness(t *testing.T) *wsHarness {
	t.Helper()
	svc := new(mockChatMessageServiceWS)
	hub := new(mockWSHub)
	return &wsHarness{svc: svc, hub: hub, wsHandler: NewWSHandler(svc, hub)}
}

func testClient() *ws.Client {
	return &ws.Client{ID: "client-1", UserID: 42, RoomID: 4}
}

func TestWsHandler_handleCreateChatMessage(t *testing.T) {
	t.Run("success: broadcasts chat message", func(t *testing.T) {
		h := newWSHarness(t)
		c := testClient()
		now := time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC)
		msg := &ChatMessage{BaseModel: model.BaseModel{ID: 1, CreatedAt: now, UpdatedAt: now}, RoomID: 4, UserID: 42, Content: "hello", IsPinned: false}

		h.svc.On("Create", &CreateChatMessageParams{RoomID: 4, UserID: 42, Content: "hello", IsPinned: false}).Return(msg, nil)
		h.hub.On("BroadcastToRoom", uint(4), ws.EventChatMessage, BuildChatMessageResponse(msg)).Return()

		h.wsHandler.handleCreateChatMessage(c, ws.Event{Payload: json.RawMessage(`{"content":"hello","isPinned":false}`)})

		h.svc.AssertExpectations(t)
		h.hub.AssertExpectations(t)
	})

	t.Run("failure: invalid payload", func(t *testing.T) {
		h := newWSHarness(t)
		c := testClient()

		h.hub.On("ErrorToClient", c, "Invalid chat message payload", http.StatusUnprocessableEntity).Return()

		h.wsHandler.handleCreateChatMessage(c, ws.Event{Payload: json.RawMessage(`{not json`)})

		h.hub.AssertExpectations(t)
		h.svc.AssertNotCalled(t, "Create", mock.Anything)
	})

	t.Run("failure: forbidden", func(t *testing.T) {
		h := newWSHarness(t)
		c := testClient()

		h.svc.On("Create", &CreateChatMessageParams{RoomID: 4, UserID: 42, Content: "hello", IsPinned: false}).
			Return(nil, httpx.ErrForbidden)
		h.hub.On("ErrorToClient", c, "Failed to create chat message", http.StatusForbidden).Return()

		h.wsHandler.handleCreateChatMessage(c, ws.Event{Payload: json.RawMessage(`{"content":"hello"}`)})

		h.svc.AssertExpectations(t)
		h.hub.AssertExpectations(t)
	})

	t.Run("failure: service error", func(t *testing.T) {
		h := newWSHarness(t)
		c := testClient()

		h.svc.On("Create", &CreateChatMessageParams{RoomID: 4, UserID: 42, Content: "hello", IsPinned: false}).
			Return(nil, assert.AnError)
		h.hub.On("ErrorToClient", c, "Failed to create chat message", http.StatusInternalServerError).Return()

		h.wsHandler.handleCreateChatMessage(c, ws.Event{Payload: json.RawMessage(`{"content":"hello"}`)})

		h.svc.AssertExpectations(t)
		h.hub.AssertExpectations(t)
	})
}
