package roomuserfavourite

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"sound-stage-backend/internal/pkg/listopts"
	"sound-stage-backend/internal/pkg/testutil"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type mockFavouriteService struct{ mock.Mock }

func (m *mockFavouriteService) Add(userID, roomID uint) error {
	args := m.Called(userID, roomID)
	return args.Error(0)
}

func (m *mockFavouriteService) Remove(userID, roomID uint) error {
	args := m.Called(userID, roomID)
	return args.Error(0)
}

func (m *mockFavouriteService) ListUserFavourites(userID uint, p listopts.Pagination) ([]RoomUserFavourite, int64, error) {
	args := m.Called(userID, p)
	favourites, _ := args.Get(0).([]RoomUserFavourite)
	return favourites, args.Get(1).(int64), args.Error(2)
}

type handlerHarness struct {
	svc     *mockFavouriteService
	handler *Handler
}

func newHandlerHarness(t *testing.T) *handlerHarness {
	t.Helper()
	svc := new(mockFavouriteService)
	return &handlerHarness{
		svc:     svc,
		handler: NewHandler(svc),
	}
}

func TestHandler_Add(t *testing.T) {
	t.Run("success: adds favourite and returns 200", func(t *testing.T) {
		h := newHandlerHarness(t)
		h.svc.On("Add", uint(20), uint(10)).Return(nil)

		w, c := testutil.NewTestContext(http.MethodPost, "/users/current/favorites", AddFavouriteInput{RoomID: 10})
		c.Set("userId", uint(20))

		h.handler.Add(c)

		assert.Equal(t, http.StatusOK, w.Code)
		h.svc.AssertExpectations(t)
	})

	t.Run("failure: malformed JSON body returns 400", func(t *testing.T) {
		h := newHandlerHarness(t)

		w, c := testutil.NewTestContext(http.MethodPost, "/users/current/favorites", "{not json")
		c.Set("userId", uint(20))

		h.handler.Add(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		h.svc.AssertNotCalled(t, "Add", mock.Anything, mock.Anything)
	})

	t.Run("failure: missing roomId returns 422", func(t *testing.T) {
		h := newHandlerHarness(t)

		w, c := testutil.NewTestContext(http.MethodPost, "/users/current/favorites", AddFavouriteInput{})
		c.Set("userId", uint(20))

		h.handler.Add(c)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		h.svc.AssertNotCalled(t, "Add", mock.Anything, mock.Anything)
	})

	t.Run("failure: service error returns 422", func(t *testing.T) {
		h := newHandlerHarness(t)
		h.svc.On("Add", uint(20), uint(10)).Return(assert.AnError)

		w, c := testutil.NewTestContext(http.MethodPost, "/users/current/favorites", AddFavouriteInput{RoomID: 10})
		c.Set("userId", uint(20))

		h.handler.Add(c)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		h.svc.AssertExpectations(t)
	})
}

func TestHandler_Remove(t *testing.T) {
	t.Run("success: removes favourite and returns 200", func(t *testing.T) {
		h := newHandlerHarness(t)
		h.svc.On("Remove", uint(20), uint(10)).Return(nil)

		w, c := testutil.NewTestContext(http.MethodDelete, "/users/current/favorites/10", nil)
		c.Params = gin.Params{{Key: "roomId", Value: "10"}}
		c.Set("userId", uint(20))

		h.handler.Remove(c)

		assert.Equal(t, http.StatusOK, w.Code)
		h.svc.AssertExpectations(t)
	})

	t.Run("failure: non-numeric roomId returns 400", func(t *testing.T) {
		h := newHandlerHarness(t)

		w, c := testutil.NewTestContext(http.MethodDelete, "/users/current/favorites/abc", nil)
		c.Params = gin.Params{{Key: "roomId", Value: "abc"}}
		c.Set("userId", uint(20))

		h.handler.Remove(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		h.svc.AssertNotCalled(t, "Remove", mock.Anything, mock.Anything)
	})

	t.Run("failure: service error returns 422", func(t *testing.T) {
		h := newHandlerHarness(t)
		h.svc.On("Remove", uint(20), uint(10)).Return(assert.AnError)

		w, c := testutil.NewTestContext(http.MethodDelete, "/users/current/favorites/10", nil)
		c.Params = gin.Params{{Key: "roomId", Value: "10"}}
		c.Set("userId", uint(20))

		h.handler.Remove(c)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		h.svc.AssertExpectations(t)
	})
}
