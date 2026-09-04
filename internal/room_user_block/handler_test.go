package roomuserblock

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"sound-stage-backend/internal/pkg/httpx"
	"sound-stage-backend/internal/pkg/testutil"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type mockBlockService struct{ mock.Mock }

func (m *mockBlockService) Add(roomID, userID, blockedByID uint) error {
	args := m.Called(roomID, userID, blockedByID)
	return args.Error(0)
}

func (m *mockBlockService) Remove(roomID, userID, blockedByID uint) error {
	args := m.Called(roomID, userID, blockedByID)
	return args.Error(0)
}

type handlerHarness struct {
	svc     *mockBlockService
	handler *Handler
}

func newHandlerHarness(t *testing.T) *handlerHarness {
	t.Helper()
	svc := new(mockBlockService)
	return &handlerHarness{
		svc:     svc,
		handler: NewHandler(svc),
	}
}

func TestHandler_Add(t *testing.T) {
	t.Run("success: blocks a user and returns 200", func(t *testing.T) {
		h := newHandlerHarness(t)
		h.svc.On("Add", uint(10), uint(20), uint(30)).Return(nil)

		w, c := testutil.NewTestContext(http.MethodPost, "/rooms/10/blocks", AddBlockInput{UserID: 20})
		c.Params = gin.Params{{Key: "id", Value: "10"}}
		c.Set("userId", uint(30))

		h.handler.Add(c)

		assert.Equal(t, http.StatusOK, w.Code)
		h.svc.AssertExpectations(t)
	})

	t.Run("failure: non-numeric room ID returns 400 before service is called", func(t *testing.T) {
		h := newHandlerHarness(t)

		w, c := testutil.NewTestContext(http.MethodPost, "/rooms/abc/blocks", AddBlockInput{UserID: 20})
		c.Params = gin.Params{{Key: "id", Value: "abc"}}
		c.Set("userId", uint(30))

		h.handler.Add(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		h.svc.AssertNotCalled(t, "Add", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("failure: malformed JSON body returns 400", func(t *testing.T) {
		h := newHandlerHarness(t)

		w, c := testutil.NewTestContext(http.MethodPost, "/rooms/10/blocks", "{not json")
		c.Params = gin.Params{{Key: "id", Value: "10"}}
		c.Set("userId", uint(30))

		h.handler.Add(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		h.svc.AssertNotCalled(t, "Add", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("failure: missing userId returns 422", func(t *testing.T) {
		h := newHandlerHarness(t)

		w, c := testutil.NewTestContext(http.MethodPost, "/rooms/10/blocks", AddBlockInput{})
		c.Params = gin.Params{{Key: "id", Value: "10"}}
		c.Set("userId", uint(30))

		h.handler.Add(c)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		h.svc.AssertNotCalled(t, "Add", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("failure: forbidden error returns 403", func(t *testing.T) {
		h := newHandlerHarness(t)
		h.svc.On("Add", uint(10), uint(20), uint(30)).Return(httpx.ErrForbidden)

		w, c := testutil.NewTestContext(http.MethodPost, "/rooms/10/blocks", AddBlockInput{UserID: 20})
		c.Params = gin.Params{{Key: "id", Value: "10"}}
		c.Set("userId", uint(30))

		h.handler.Add(c)

		assert.Equal(t, http.StatusForbidden, w.Code)
		h.svc.AssertExpectations(t)
	})

	t.Run("failure: service error returns 422", func(t *testing.T) {
		h := newHandlerHarness(t)
		h.svc.On("Add", uint(10), uint(20), uint(30)).Return(assert.AnError)

		w, c := testutil.NewTestContext(http.MethodPost, "/rooms/10/blocks", AddBlockInput{UserID: 20})
		c.Params = gin.Params{{Key: "id", Value: "10"}}
		c.Set("userId", uint(30))

		h.handler.Add(c)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		h.svc.AssertExpectations(t)
	})
}

func TestHandler_Remove(t *testing.T) {
	t.Run("success: unblocks a user and returns 200", func(t *testing.T) {
		h := newHandlerHarness(t)
		h.svc.On("Remove", uint(10), uint(20), uint(30)).Return(nil)

		w, c := testutil.NewTestContext(http.MethodDelete, "/rooms/10/blocks/20", nil)
		c.Params = gin.Params{{Key: "id", Value: "10"}, {Key: "userId", Value: "20"}}
		c.Set("userId", uint(30))

		h.handler.Remove(c)

		assert.Equal(t, http.StatusOK, w.Code)
		h.svc.AssertExpectations(t)
	})

	t.Run("failure: non-numeric room ID returns 400", func(t *testing.T) {
		h := newHandlerHarness(t)

		w, c := testutil.NewTestContext(http.MethodDelete, "/rooms/abc/blocks/20", nil)
		c.Params = gin.Params{{Key: "id", Value: "abc"}, {Key: "userId", Value: "20"}}
		c.Set("userId", uint(30))

		h.handler.Remove(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		h.svc.AssertNotCalled(t, "Remove", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("failure: non-numeric user ID returns 400", func(t *testing.T) {
		h := newHandlerHarness(t)

		w, c := testutil.NewTestContext(http.MethodDelete, "/rooms/10/blocks/abc", nil)
		c.Params = gin.Params{{Key: "id", Value: "10"}, {Key: "userId", Value: "abc"}}
		c.Set("userId", uint(30))

		h.handler.Remove(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		h.svc.AssertNotCalled(t, "Remove", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("failure: forbidden error returns 403", func(t *testing.T) {
		h := newHandlerHarness(t)
		h.svc.On("Remove", uint(10), uint(20), uint(30)).Return(httpx.ErrForbidden)

		w, c := testutil.NewTestContext(http.MethodDelete, "/rooms/10/blocks/20", nil)
		c.Params = gin.Params{{Key: "id", Value: "10"}, {Key: "userId", Value: "20"}}
		c.Set("userId", uint(30))

		h.handler.Remove(c)

		assert.Equal(t, http.StatusForbidden, w.Code)
		h.svc.AssertExpectations(t)
	})

	t.Run("failure: service error returns 422", func(t *testing.T) {
		h := newHandlerHarness(t)
		h.svc.On("Remove", uint(10), uint(20), uint(30)).Return(assert.AnError)

		w, c := testutil.NewTestContext(http.MethodDelete, "/rooms/10/blocks/20", nil)
		c.Params = gin.Params{{Key: "id", Value: "10"}, {Key: "userId", Value: "20"}}
		c.Set("userId", uint(30))

		h.handler.Remove(c)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		h.svc.AssertExpectations(t)
	})
}
