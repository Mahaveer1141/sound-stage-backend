package chatmessage

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"sound-stage-backend/internal/pkg/httpx"
	"sound-stage-backend/internal/pkg/listopts"
	"sound-stage-backend/internal/pkg/testutil"
	"sound-stage-backend/internal/ws"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type mockChatMessageService struct{ mock.Mock }

func (m *mockChatMessageService) Create(input *CreateChatMessageParams) (*ChatMessage, error) {
	args := m.Called(input)
	msg, _ := args.Get(0).(*ChatMessage)
	return msg, args.Error(1)
}

func (m *mockChatMessageService) List(userID uint, filter ChatMessageFilter, p listopts.Pagination) ([]ChatMessage, int64, error) {
	args := m.Called(userID, filter, p)
	msgs, _ := args.Get(0).([]ChatMessage)
	return msgs, args.Get(1).(int64), args.Error(2)
}

func (m *mockChatMessageService) SetPinned(userID, roomID, messageID uint, pinned bool) (*ChatMessage, error) {
	args := m.Called(userID, roomID, messageID, pinned)
	msg, _ := args.Get(0).(*ChatMessage)
	return msg, args.Error(1)
}

type mockWSHub struct{ mock.Mock }

func (m *mockWSHub) BroadcastToRoom(roomID uint, eventName ws.EventName, payload any) {
	m.Called(roomID, eventName, payload)
}

type handlerHarness struct {
	svc     *mockChatMessageService
	hub     *mockWSHub
	handler *Handler
}

func newHandlerHarness(t *testing.T) *handlerHarness {
	t.Helper()
	svc := new(mockChatMessageService)
	hub := new(mockWSHub)
	return &handlerHarness{svc: svc, hub: hub, handler: NewHandler(svc, hub)}
}

func TestHandler_Create(t *testing.T) {
	newCreateContext := func(target string, body any) (*httptest.ResponseRecorder, *gin.Context) {
		w, c := testutil.NewTestContext(http.MethodPost, target, body)
		c.Set("userId", uint(42))
		return w, c
	}

	t.Run("success: creates and broadcasts a message", func(t *testing.T) {
		h := newHandlerHarness(t)
		msg := &ChatMessage{RoomID: 5, UserID: 42, Content: "hello"}
		input := &CreateChatMessageParams{RoomID: 5, UserID: 42, Content: "hello"}

		h.svc.On("Create", input).Return(msg, nil)
		h.hub.On("BroadcastToRoom", uint(5), ws.EventChatMessage, BuildChatMessageResponse(msg)).Return()

		w, c := newCreateContext("/rooms/5/messages", createMessageRequest{Content: "hello"})
		c.Params = gin.Params{{Key: "id", Value: "5"}}

		h.handler.Create(c)

		assert.Equal(t, http.StatusCreated, w.Code)
		h.svc.AssertExpectations(t)
		h.hub.AssertExpectations(t)
	})

	t.Run("failure: invalid room ID", func(t *testing.T) {
		h := newHandlerHarness(t)

		w, c := newCreateContext("/rooms/abc/messages", createMessageRequest{Content: "hello"})
		c.Params = gin.Params{{Key: "id", Value: "abc"}}

		h.handler.Create(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		h.svc.AssertNotCalled(t, "Create", mock.Anything)
	})

	t.Run("failure: invalid body", func(t *testing.T) {
		h := newHandlerHarness(t)

		w, c := newCreateContext("/rooms/5/messages", "not json")
		c.Params = gin.Params{{Key: "id", Value: "5"}}

		h.handler.Create(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		h.svc.AssertNotCalled(t, "Create", mock.Anything)
	})

	t.Run("failure: empty content", func(t *testing.T) {
		h := newHandlerHarness(t)

		w, c := newCreateContext("/rooms/5/messages", createMessageRequest{Content: ""})
		c.Params = gin.Params{{Key: "id", Value: "5"}}

		h.handler.Create(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		h.svc.AssertNotCalled(t, "Create", mock.Anything)
	})

	t.Run("failure: forbidden", func(t *testing.T) {
		h := newHandlerHarness(t)
		h.svc.On("Create", mock.Anything).Return(nil, httpx.ErrForbidden)

		w, c := newCreateContext("/rooms/5/messages", createMessageRequest{Content: "hello"})
		c.Params = gin.Params{{Key: "id", Value: "5"}}

		h.handler.Create(c)

		assert.Equal(t, http.StatusForbidden, w.Code)
		h.hub.AssertNotCalled(t, "BroadcastToRoom", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("failure: service error", func(t *testing.T) {
		h := newHandlerHarness(t)
		h.svc.On("Create", mock.Anything).Return(nil, assert.AnError)

		w, c := newCreateContext("/rooms/5/messages", createMessageRequest{Content: "hello"})
		c.Params = gin.Params{{Key: "id", Value: "5"}}

		h.handler.Create(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		h.hub.AssertNotCalled(t, "BroadcastToRoom", mock.Anything, mock.Anything, mock.Anything)
	})
}

func TestHandler_List(t *testing.T) {
	t.Run("success: returns paginated messages", func(t *testing.T) {
		h := newHandlerHarness(t)
		messages := []ChatMessage{{Content: "hi"}}

		h.svc.On("List", uint(42), ChatMessageFilter{RoomID: 5}, listopts.Pagination{Page: 1, PageSize: 10}).
			Return(messages, int64(1), nil)

		w, c := testutil.NewTestContext(http.MethodGet, "/rooms/5/chat-messages?page=1&pageSize=10", nil)
		c.Params = gin.Params{{Key: "id", Value: "5"}}
		c.Set("userId", uint(42))

		h.handler.List(c)

		assert.Equal(t, http.StatusOK, w.Code)
		h.svc.AssertExpectations(t)
	})

	t.Run("failure: invalid room ID", func(t *testing.T) {
		h := newHandlerHarness(t)

		w, c := testutil.NewTestContext(http.MethodGet, "/rooms/abc/chat-messages?page=1&pageSize=10", nil)
		c.Params = gin.Params{{Key: "id", Value: "abc"}}
		c.Set("userId", uint(42))

		h.handler.List(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		h.svc.AssertNotCalled(t, "List", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("failure: invalid pagination", func(t *testing.T) {
		h := newHandlerHarness(t)

		w, c := testutil.NewTestContext(http.MethodGet, "/rooms/5/chat-messages?page=0&pageSize=10", nil)
		c.Params = gin.Params{{Key: "id", Value: "5"}}
		c.Set("userId", uint(42))

		h.handler.List(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		h.svc.AssertNotCalled(t, "List", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("failure: forbidden", func(t *testing.T) {
		h := newHandlerHarness(t)

		h.svc.On("List", uint(42), ChatMessageFilter{RoomID: 5}, listopts.Pagination{Page: 1, PageSize: 10}).
			Return(nil, int64(0), httpx.ErrForbidden)

		w, c := testutil.NewTestContext(http.MethodGet, "/rooms/5/chat-messages?page=1&pageSize=10", nil)
		c.Params = gin.Params{{Key: "id", Value: "5"}}
		c.Set("userId", uint(42))

		h.handler.List(c)

		assert.Equal(t, http.StatusForbidden, w.Code)
		h.svc.AssertExpectations(t)
	})

	t.Run("failure: service error", func(t *testing.T) {
		h := newHandlerHarness(t)

		h.svc.On("List", uint(42), ChatMessageFilter{RoomID: 5}, listopts.Pagination{Page: 1, PageSize: 10}).
			Return(nil, int64(0), assert.AnError)

		w, c := testutil.NewTestContext(http.MethodGet, "/rooms/5/chat-messages?page=1&pageSize=10", nil)
		c.Params = gin.Params{{Key: "id", Value: "5"}}
		c.Set("userId", uint(42))

		h.handler.List(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		h.svc.AssertExpectations(t)
	})
}

func TestHandler_SetPinned(t *testing.T) {
	newSetPinnedContext := func(target string, body any) (*httptest.ResponseRecorder, *gin.Context) {
		w, c := testutil.NewTestContext(http.MethodPatch, target, body)
		c.Set("userId", uint(42))
		return w, c
	}

	t.Run("success: unpins a message", func(t *testing.T) {
		h := newHandlerHarness(t)
		h.svc.On("SetPinned", uint(42), uint(5), uint(9), false).
			Return(&ChatMessage{Content: "hi"}, nil)

		w, c := newSetPinnedContext("/rooms/5/messages/9/pin", setPinnedRequest{IsPinned: false})
		c.Params = gin.Params{{Key: "id", Value: "5"}, {Key: "messageId", Value: "9"}}

		h.handler.SetPinned(c)

		assert.Equal(t, http.StatusOK, w.Code)
		h.svc.AssertExpectations(t)
	})

	t.Run("failure: invalid room ID", func(t *testing.T) {
		h := newHandlerHarness(t)

		w, c := newSetPinnedContext("/rooms/abc/messages/9/pin", setPinnedRequest{IsPinned: false})
		c.Params = gin.Params{{Key: "id", Value: "abc"}, {Key: "messageId", Value: "9"}}

		h.handler.SetPinned(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		h.svc.AssertNotCalled(t, "SetPinned", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("failure: invalid message ID", func(t *testing.T) {
		h := newHandlerHarness(t)

		w, c := newSetPinnedContext("/rooms/5/messages/abc/pin", setPinnedRequest{IsPinned: false})
		c.Params = gin.Params{{Key: "id", Value: "5"}, {Key: "messageId", Value: "abc"}}

		h.handler.SetPinned(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		h.svc.AssertNotCalled(t, "SetPinned", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("failure: invalid body", func(t *testing.T) {
		h := newHandlerHarness(t)

		w, c := newSetPinnedContext("/rooms/5/messages/9/pin", "not json")
		c.Params = gin.Params{{Key: "id", Value: "5"}, {Key: "messageId", Value: "9"}}

		h.handler.SetPinned(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		h.svc.AssertNotCalled(t, "SetPinned", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("failure: forbidden", func(t *testing.T) {
		h := newHandlerHarness(t)
		h.svc.On("SetPinned", uint(42), uint(5), uint(9), false).
			Return(nil, httpx.ErrForbidden)

		w, c := newSetPinnedContext("/rooms/5/messages/9/pin", setPinnedRequest{IsPinned: false})
		c.Params = gin.Params{{Key: "id", Value: "5"}, {Key: "messageId", Value: "9"}}

		h.handler.SetPinned(c)

		assert.Equal(t, http.StatusForbidden, w.Code)
		h.svc.AssertExpectations(t)
	})

	t.Run("failure: message not found", func(t *testing.T) {
		h := newHandlerHarness(t)
		h.svc.On("SetPinned", uint(42), uint(5), uint(9), false).
			Return(nil, httpx.ErrRecordNotFound)

		w, c := newSetPinnedContext("/rooms/5/messages/9/pin", setPinnedRequest{IsPinned: false})
		c.Params = gin.Params{{Key: "id", Value: "5"}, {Key: "messageId", Value: "9"}}

		h.handler.SetPinned(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
		h.svc.AssertExpectations(t)
	})

	t.Run("failure: pin limit reached", func(t *testing.T) {
		h := newHandlerHarness(t)
		h.svc.On("SetPinned", uint(42), uint(5), uint(9), true).
			Return(nil, httpx.ErrPinnedLimitReached)

		w, c := newSetPinnedContext("/rooms/5/messages/9/pin", setPinnedRequest{IsPinned: true})
		c.Params = gin.Params{{Key: "id", Value: "5"}, {Key: "messageId", Value: "9"}}

		h.handler.SetPinned(c)

		assert.Equal(t, http.StatusConflict, w.Code)
		h.svc.AssertExpectations(t)
	})

	t.Run("failure: service error", func(t *testing.T) {
		h := newHandlerHarness(t)
		h.svc.On("SetPinned", uint(42), uint(5), uint(9), false).
			Return(nil, assert.AnError)

		w, c := newSetPinnedContext("/rooms/5/messages/9/pin", setPinnedRequest{IsPinned: false})
		c.Params = gin.Params{{Key: "id", Value: "5"}, {Key: "messageId", Value: "9"}}

		h.handler.SetPinned(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		h.svc.AssertExpectations(t)
	})
}
