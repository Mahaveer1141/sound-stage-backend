package roomuser

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"sound-stage-backend/internal/pkg/httpx"
	"sound-stage-backend/internal/pkg/listopts"
	"sound-stage-backend/internal/pkg/testutil"
	"sound-stage-backend/internal/role"
	"sound-stage-backend/internal/ws"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type mockRoomUserService struct{ mock.Mock }

func (m *mockRoomUserService) ListByRoomID(roomID uint, filter RoomUserFilter, sort listopts.Sort, p listopts.Pagination) ([]RoomUser, int64, error) {
	args := m.Called(roomID, filter, sort, p)
	rus, _ := args.Get(0).([]RoomUser)
	return rus, args.Get(1).(int64), args.Error(2)
}
func (m *mockRoomUserService) FindBy(userID, roomID uint) (*RoomUser, error) {
	args := m.Called(userID, roomID)
	ru, _ := args.Get(0).(*RoomUser)
	return ru, args.Error(1)
}
func (m *mockRoomUserService) UpdateRole(roomID, userID uint, roleName role.RoleName, actorID uint) error {
	args := m.Called(roomID, userID, roleName, actorID)
	return args.Error(0)
}

type mockWebSocketHub struct{ mock.Mock }

func (m *mockWebSocketHub) BroadcastToRoom(roomID uint, eventName ws.EventName, payload any) {
	m.Called(roomID, eventName, payload)
}

type mockRoomJoiner struct{ mock.Mock }

func (m *mockRoomJoiner) AddRoomUser(roomID, userID uint, privateCode string) (*RoomUser, error) {
	args := m.Called(roomID, userID, privateCode)
	ru, _ := args.Get(0).(*RoomUser)
	return ru, args.Error(1)
}

type handlerHarness struct {
	svc     *mockRoomUserService
	joiner  *mockRoomJoiner
	hub     *mockWebSocketHub
	handler *Handler
}

func newHandlerHarness(t *testing.T) *handlerHarness {
	t.Helper()
	svc := new(mockRoomUserService)
	joiner := new(mockRoomJoiner)
	hub := new(mockWebSocketHub)
	return &handlerHarness{
		svc:     svc,
		joiner:  joiner,
		hub:     hub,
		handler: NewHandler(svc, joiner, hub),
	}
}

func TestHandler_ListUsers(t *testing.T) {
	t.Run("success: returns 200 with mapped user responses", func(t *testing.T) {
		h := newHandlerHarness(t)
		users := []RoomUser{{UserID: 1}, {UserID: 2}}
		h.svc.On("ListByRoomID", uint(4), mock.Anything, mock.Anything, mock.Anything).Return(users, int64(2), nil)
		h.svc.On("FindBy", uint(1), uint(4)).Return(&RoomUser{Role: role.Role{Name: string(role.RoleListener)}}, nil)

		w, c := testutil.NewTestContext(http.MethodGet, "/rooms/4/users?page=1&pageSize=10", nil)
		c.Params = gin.Params{{Key: "id", Value: "4"}}
		c.Set("userId", uint(1))

		h.handler.ListUsers(c)

		assert.Equal(t, http.StatusOK, w.Code)
		h.svc.AssertExpectations(t)
	})

	t.Run("failure: non-numeric room ID returns 400 before service is called", func(t *testing.T) {
		h := newHandlerHarness(t)

		w, c := testutil.NewTestContext(http.MethodGet, "/rooms/abc/users?page=1&pageSize=10", nil)
		c.Params = gin.Params{{Key: "id", Value: "abc"}}

		h.handler.ListUsers(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		h.svc.AssertNotCalled(t, "ListByRoomID", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("failure: service error returns 500", func(t *testing.T) {
		h := newHandlerHarness(t)
		h.svc.On("ListByRoomID", uint(4), mock.Anything, mock.Anything, mock.Anything).Return(nil, int64(0), assert.AnError)

		w, c := testutil.NewTestContext(http.MethodGet, "/rooms/4/users?page=1&pageSize=10", nil)
		c.Params = gin.Params{{Key: "id", Value: "4"}}

		h.handler.ListUsers(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		h.svc.AssertExpectations(t)
	})
}

func TestHandler_CurrentRoomUser(t *testing.T) {
	t.Run("success: returns 200 with current room user", func(t *testing.T) {
		h := newHandlerHarness(t)
		want := &RoomUser{UserID: 1, RoomID: 4}
		h.svc.On("FindBy", uint(1), uint(4)).Return(want, nil)

		w, c := testutil.NewTestContext(http.MethodGet, "/rooms/4/users/current", nil)
		c.Params = gin.Params{{Key: "id", Value: "4"}}
		c.Set("userId", uint(1))

		h.handler.CurrentRoomUser(c)

		assert.Equal(t, http.StatusOK, w.Code)
		h.svc.AssertExpectations(t)
	})

	t.Run("failure: non-numeric room ID returns 400 before service is called", func(t *testing.T) {
		h := newHandlerHarness(t)

		w, c := testutil.NewTestContext(http.MethodGet, "/rooms/abc/users/current", nil)
		c.Params = gin.Params{{Key: "id", Value: "abc"}}

		h.handler.CurrentRoomUser(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		h.svc.AssertNotCalled(t, "FindBy", mock.Anything, mock.Anything)
	})

	t.Run("failure: nil result (not a member) returns 422", func(t *testing.T) {
		h := newHandlerHarness(t)
		h.svc.On("FindBy", uint(1), uint(4)).Return(nil, nil)

		w, c := testutil.NewTestContext(http.MethodGet, "/rooms/4/users/current", nil)
		c.Params = gin.Params{{Key: "id", Value: "4"}}
		c.Set("userId", uint(1))

		h.handler.CurrentRoomUser(c)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		h.svc.AssertExpectations(t)
	})
}

func TestHandler_UpdateUserRole(t *testing.T) {
	t.Run("success: updates role, broadcasts, returns 200", func(t *testing.T) {
		h := newHandlerHarness(t)
		h.svc.On("UpdateRole", uint(4), uint(7), role.RoleSpeaker, uint(1)).Return(nil)
		h.hub.On("BroadcastToRoom", uint(4), ws.EventUserRoleUpdated, mock.MatchedBy(func(payload any) bool {
			gh, ok := payload.(gin.H)
			return ok && gh["userId"] == 7 && gh["role"] == role.RoleSpeaker
		})).Return()

		w, c := testutil.NewTestContext(http.MethodPatch, "/rooms/4/users/7/role", updateUserRoleInput{Role: role.RoleSpeaker})
		c.Params = gin.Params{{Key: "id", Value: "4"}, {Key: "userId", Value: "7"}}
		c.Set("userId", uint(1))

		h.handler.UpdateUserRole(c)

		assert.Equal(t, http.StatusOK, w.Code)
		h.svc.AssertExpectations(t)
		h.hub.AssertExpectations(t)
	})

	t.Run("failure: forbidden error from service returns 403, no broadcast", func(t *testing.T) {
		h := newHandlerHarness(t)
		h.svc.On("UpdateRole", uint(4), uint(7), role.RoleSpeaker, uint(1)).Return(httpx.ErrForbidden)

		w, c := testutil.NewTestContext(http.MethodPatch, "/rooms/4/users/7/role", updateUserRoleInput{Role: role.RoleSpeaker})
		c.Params = gin.Params{{Key: "id", Value: "4"}, {Key: "userId", Value: "7"}}
		c.Set("userId", uint(1))

		h.handler.UpdateUserRole(c)

		assert.Equal(t, http.StatusForbidden, w.Code)
		h.svc.AssertExpectations(t)
		h.hub.AssertNotCalled(t, "BroadcastToRoom", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("failure: non-numeric room ID returns 400 before service is called", func(t *testing.T) {
		h := newHandlerHarness(t)

		w, c := testutil.NewTestContext(http.MethodPatch, "/rooms/abc/users/7/role", updateUserRoleInput{Role: role.RoleSpeaker})
		c.Params = gin.Params{{Key: "id", Value: "abc"}, {Key: "userId", Value: "7"}}
		c.Set("userId", uint(1))

		h.handler.UpdateUserRole(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		h.svc.AssertNotCalled(t, "UpdateRole", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("failure: non-numeric user ID returns 400 before service is called", func(t *testing.T) {
		h := newHandlerHarness(t)

		w, c := testutil.NewTestContext(http.MethodPatch, "/rooms/4/users/abc/role", updateUserRoleInput{Role: role.RoleSpeaker})
		c.Params = gin.Params{{Key: "id", Value: "4"}, {Key: "userId", Value: "abc"}}
		c.Set("userId", uint(1))

		h.handler.UpdateUserRole(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		h.svc.AssertNotCalled(t, "UpdateRole", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("failure: other service error returns 422", func(t *testing.T) {
		h := newHandlerHarness(t)
		h.svc.On("UpdateRole", uint(4), uint(7), role.RoleSpeaker, uint(1)).Return(assert.AnError)

		w, c := testutil.NewTestContext(http.MethodPatch, "/rooms/4/users/7/role", updateUserRoleInput{Role: role.RoleSpeaker})
		c.Params = gin.Params{{Key: "id", Value: "4"}, {Key: "userId", Value: "7"}}
		c.Set("userId", uint(1))

		h.handler.UpdateUserRole(c)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		h.svc.AssertExpectations(t)
	})
}

func TestHandler_AddRoomUser(t *testing.T) {
	t.Run("success: adds user to room and returns 200", func(t *testing.T) {
		h := newHandlerHarness(t)
		want := &RoomUser{UserID: 1, RoomID: 4}
		h.joiner.On("AddRoomUser", uint(4), uint(1), "secret").Return(want, nil)

		w, c := testutil.NewTestContext(http.MethodPost, "/rooms/4/users", addRoomUserInput{PrivateCode: "secret"})
		c.Params = gin.Params{{Key: "id", Value: "4"}}
		c.Set("userId", uint(1))

		h.handler.AddRoomUser(c)

		assert.Equal(t, http.StatusOK, w.Code)
		h.joiner.AssertExpectations(t)
	})

	t.Run("failure: non-numeric room ID returns 400 before service is called", func(t *testing.T) {
		h := newHandlerHarness(t)

		w, c := testutil.NewTestContext(http.MethodPost, "/rooms/abc/users", addRoomUserInput{})
		c.Params = gin.Params{{Key: "id", Value: "abc"}}
		c.Set("userId", uint(1))

		h.handler.AddRoomUser(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		h.joiner.AssertNotCalled(t, "AddRoomUser", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("failure: incorrect private code returns 403", func(t *testing.T) {
		h := newHandlerHarness(t)
		h.joiner.On("AddRoomUser", uint(4), uint(1), "wrong").Return(nil, httpx.ErrForbidden)

		w, c := testutil.NewTestContext(http.MethodPost, "/rooms/4/users", addRoomUserInput{PrivateCode: "wrong"})
		c.Params = gin.Params{{Key: "id", Value: "4"}}
		c.Set("userId", uint(1))

		h.handler.AddRoomUser(c)

		assert.Equal(t, http.StatusForbidden, w.Code)
		h.joiner.AssertExpectations(t)
	})

	t.Run("failure: service error returns 422", func(t *testing.T) {
		h := newHandlerHarness(t)
		h.joiner.On("AddRoomUser", uint(4), uint(1), "").Return(nil, assert.AnError)

		w, c := testutil.NewTestContext(http.MethodPost, "/rooms/4/users", addRoomUserInput{})
		c.Params = gin.Params{{Key: "id", Value: "4"}}
		c.Set("userId", uint(1))

		h.handler.AddRoomUser(c)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		h.joiner.AssertExpectations(t)
	})
}
