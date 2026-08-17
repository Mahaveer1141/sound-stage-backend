package user

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"sound-stage-backend/internal/pkg/testutil"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type mockUserService struct{ mock.Mock }

func (m *mockUserService) FindByID(userId uint) (*User, error) {
	args := m.Called(userId)
	u, _ := args.Get(0).(*User)
	return u, args.Error(1)
}
func (m *mockUserService) UpdateProfile(id uint, input *UpdateUserParams) (*User, error) {
	args := m.Called(id, input)
	u, _ := args.Get(0).(*User)
	return u, args.Error(1)
}

type handlerHarness struct {
	svc *mockUserService
	h   *Handler
}

func newHandlerHarness(t *testing.T) *handlerHarness {
	t.Helper()
	svc := new(mockUserService)
	return &handlerHarness{
		svc: svc,
		h:   NewHandler(svc),
	}
}

func TestHandler_CurrentUser(t *testing.T) {
	t.Run("success: returns 200 with current user", func(t *testing.T) {
		h := newHandlerHarness(t)
		want := &User{Email: "a@example.com"}
		h.svc.On("FindByID", uint(42)).Return(want, nil)

		w, c := testutil.NewTestContext(http.MethodGet, "/users/me", nil)
		c.Set("userId", uint(42))

		h.h.CurrentUser(c)

		assert.Equal(t, http.StatusOK, w.Code)
		h.svc.AssertExpectations(t)
	})

	t.Run("failure: service error returns 500", func(t *testing.T) {
		h := newHandlerHarness(t)
		h.svc.On("FindByID", uint(42)).Return(nil, assert.AnError)

		w, c := testutil.NewTestContext(http.MethodGet, "/users/me", nil)
		c.Set("userId", uint(42))

		h.h.CurrentUser(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		h.svc.AssertExpectations(t)
	})
}

func TestHandler_UpdateProfile(t *testing.T) {
	t.Run("success: updates profile and returns 200", func(t *testing.T) {
		h := newHandlerHarness(t)
		updated := &User{FirstName: "Grace"}
		h.svc.On("UpdateProfile", uint(42), mock.MatchedBy(func(in *UpdateUserParams) bool {
			return in.FirstName == "Grace"
		})).Return(updated, nil)

		w, c := testutil.NewTestContext(http.MethodPatch, "/users/me", UpdateUserParams{FirstName: "Grace"})
		c.Set("userId", uint(42))

		h.h.UpdateProfile(c)

		assert.Equal(t, http.StatusOK, w.Code)
		h.svc.AssertExpectations(t)
	})

	t.Run("failure: malformed JSON body returns 400 before service is called", func(t *testing.T) {
		h := newHandlerHarness(t)

		w, c := testutil.NewTestContext(http.MethodPatch, "/users/me", "{not json")
		c.Set("userId", uint(42))

		h.h.UpdateProfile(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		h.svc.AssertNotCalled(t, "UpdateProfile", mock.Anything, mock.Anything)
	})

	t.Run("failure: service error returns 500", func(t *testing.T) {
		h := newHandlerHarness(t)
		h.svc.On("UpdateProfile", uint(42), mock.Anything).Return(nil, assert.AnError)

		w, c := testutil.NewTestContext(http.MethodPatch, "/users/me", UpdateUserParams{FirstName: "Grace"})
		c.Set("userId", uint(42))

		h.h.UpdateProfile(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		h.svc.AssertExpectations(t)
	})
}
