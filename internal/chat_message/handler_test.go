package chatmessage

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"sound-stage-backend/internal/pkg/httpx"
	"sound-stage-backend/internal/pkg/listopts"
	"sound-stage-backend/internal/pkg/testutil"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type mockChatMessageService struct{ mock.Mock }

func (m *mockChatMessageService) List(userID uint, filter ChatMessageFilter, p listopts.Pagination) ([]ChatMessage, int64, error) {
	args := m.Called(userID, filter, p)
	msgs, _ := args.Get(0).([]ChatMessage)
	return msgs, args.Get(1).(int64), args.Error(2)
}

type handlerHarness struct {
	svc     *mockChatMessageService
	handler *Handler
}

func newHandlerHarness(t *testing.T) *handlerHarness {
	t.Helper()
	svc := new(mockChatMessageService)
	return &handlerHarness{svc: svc, handler: NewHandler(svc)}
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
