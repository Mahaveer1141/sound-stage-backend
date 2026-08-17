package room

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
	roomuser "sound-stage-backend/internal/room_user"
	"sound-stage-backend/internal/ws"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type mockRoomService struct{ mock.Mock }

func (m *mockRoomService) List(filter RoomFilter, sort listopts.Sort, p listopts.Pagination) ([]Room, int64, error) {
	args := m.Called(filter, sort, p)
	rooms, _ := args.Get(0).([]Room)
	return rooms, args.Get(1).(int64), args.Error(2)
}
func (m *mockRoomService) FindByID(id uint) (*Room, error) {
	args := m.Called(id)
	r, _ := args.Get(0).(*Room)
	return r, args.Error(1)
}
func (m *mockRoomService) Create(input *CreateRoomParams) (*Room, error) {
	args := m.Called(input)
	r, _ := args.Get(0).(*Room)
	return r, args.Error(1)
}
func (m *mockRoomService) Update(id uint, input *UpdateRoomParams) (*Room, error) {
	args := m.Called(id, input)
	r, _ := args.Get(0).(*Room)
	return r, args.Error(1)
}
func (m *mockRoomService) ListUsers(roomID uint, filter roomuser.RoomUserFilter, sort listopts.Sort, p listopts.Pagination) ([]roomuser.RoomUser, int64, error) {
	args := m.Called(roomID, filter, sort, p)
	rus, _ := args.Get(0).([]roomuser.RoomUser)
	return rus, args.Get(1).(int64), args.Error(2)
}
func (m *mockRoomService) UpdateUserRole(roomID, userID uint, newRole role.RoleName, actorID uint) error {
	args := m.Called(roomID, userID, newRole, actorID)
	return args.Error(0)
}
func (m *mockRoomService) CurrentRoomUser(roomID, userID uint) (*roomuser.RoomUser, error) {
	args := m.Called(roomID, userID)
	ru, _ := args.Get(0).(*roomuser.RoomUser)
	return ru, args.Error(1)
}

type mockWebSocketBroadcaster struct{ mock.Mock }

func (m *mockWebSocketBroadcaster) BroadcastToRoom(roomID uint, eventName ws.EventName, payload any) {
	m.Called(roomID, eventName, payload)
}

type handlerHarness struct {
	svc     *mockRoomService
	hub     *mockWebSocketBroadcaster
	handler *Handler
}

func newHandlerHarness(t *testing.T) *handlerHarness {
	t.Helper()
	svc := new(mockRoomService)
	hub := new(mockWebSocketBroadcaster)
	return &handlerHarness{
		svc:     svc,
		hub:     hub,
		handler: NewHandler(svc, hub),
	}
}

func TestHandler_Create(t *testing.T) {
	t.Run("success: creates room and returns 200", func(t *testing.T) {
		h := newHandlerHarness(t)
		created := &Room{Name: "Main Stage"}
		h.svc.On("Create", mock.MatchedBy(func(in *CreateRoomParams) bool {
			return in.Name == "Main Stage" && in.CreatorID == 42
		})).Return(created, nil)

		w, c := testutil.NewTestContext(http.MethodPost, "/rooms", CreateRoomParams{Name: "Main Stage"})
		c.Set("userId", uint(42))

		h.handler.Create(c)

		assert.Equal(t, http.StatusOK, w.Code)
		h.svc.AssertExpectations(t)
	})

	t.Run("failure: service error returns 422", func(t *testing.T) {
		h := newHandlerHarness(t)
		h.svc.On("Create", mock.Anything).Return(nil, assert.AnError)

		w, c := testutil.NewTestContext(http.MethodPost, "/rooms", CreateRoomParams{Name: "Main Stage"})
		c.Set("userId", uint(42))

		h.handler.Create(c)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		h.svc.AssertExpectations(t)
	})

	t.Run("failure: malformed JSON body returns 400 before service is called", func(t *testing.T) {
		h := newHandlerHarness(t)

		w, c := testutil.NewTestContext(http.MethodPost, "/rooms", "{not json")
		c.Set("userId", uint(42))

		h.handler.Create(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		h.svc.AssertNotCalled(t, "Create", mock.Anything)
	})
}

func TestHandler_Update(t *testing.T) {
	t.Run("success: updates room and returns 200", func(t *testing.T) {
		h := newHandlerHarness(t)
		updated := &Room{Name: "Renamed"}
		h.svc.On("Update", uint(5), mock.AnythingOfType("*room.UpdateRoomParams")).Return(updated, nil)

		w, c := testutil.NewTestContext(http.MethodPatch, "/rooms/5", UpdateRoomParams{Name: "Renamed"})
		c.Params = gin.Params{{Key: "id", Value: "5"}}

		h.handler.Update(c)

		assert.Equal(t, http.StatusOK, w.Code)
		h.svc.AssertExpectations(t)
	})

	t.Run("failure: non-numeric room ID returns 400 before service is called", func(t *testing.T) {
		h := newHandlerHarness(t)

		w, c := testutil.NewTestContext(http.MethodPatch, "/rooms/abc", UpdateRoomParams{Name: "Renamed"})
		c.Params = gin.Params{{Key: "id", Value: "abc"}}

		h.handler.Update(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		h.svc.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	})

	t.Run("failure: service error returns 422", func(t *testing.T) {
		h := newHandlerHarness(t)
		h.svc.On("Update", uint(5), mock.Anything).Return(nil, assert.AnError)

		w, c := testutil.NewTestContext(http.MethodPatch, "/rooms/5", UpdateRoomParams{Name: "Renamed"})
		c.Params = gin.Params{{Key: "id", Value: "5"}}

		h.handler.Update(c)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		h.svc.AssertExpectations(t)
	})
}

func TestHandler_List(t *testing.T) {
	t.Run("success: returns 200 with paginated rooms", func(t *testing.T) {
		h := newHandlerHarness(t)
		rooms := []Room{{Name: "A"}, {Name: "B"}}
		h.svc.On("List", mock.Anything, mock.Anything, mock.Anything).Return(rooms, int64(2), nil)

		w, c := testutil.NewTestContext(http.MethodGet, "/rooms?page=1&pageSize=10", nil)

		h.handler.List(c)

		assert.Equal(t, http.StatusOK, w.Code)
		h.svc.AssertExpectations(t)
	})

	t.Run("failure: non-positive page returns 400 before service is called", func(t *testing.T) {
		h := newHandlerHarness(t)

		w, c := testutil.NewTestContext(http.MethodGet, "/rooms?page=0&pageSize=10", nil)

		h.handler.List(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		h.svc.AssertNotCalled(t, "List", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("failure: service error returns 500", func(t *testing.T) {
		h := newHandlerHarness(t)
		h.svc.On("List", mock.Anything, mock.Anything, mock.Anything).Return(nil, int64(0), assert.AnError)

		w, c := testutil.NewTestContext(http.MethodGet, "/rooms?page=1&pageSize=10", nil)

		h.handler.List(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		h.svc.AssertExpectations(t)
	})
}

func TestHandler_FindByID(t *testing.T) {
	t.Run("success: returns 200 with room", func(t *testing.T) {
		h := newHandlerHarness(t)
		want := &Room{Name: "Main Stage"}
		h.svc.On("FindByID", uint(7)).Return(want, nil)

		w, c := testutil.NewTestContext(http.MethodGet, "/rooms/7", nil)
		c.Params = gin.Params{{Key: "id", Value: "7"}}

		h.handler.FindByID(c)

		assert.Equal(t, http.StatusOK, w.Code)
		h.svc.AssertExpectations(t)
	})

	t.Run("failure: non-numeric ID returns 400 before service is called", func(t *testing.T) {
		h := newHandlerHarness(t)

		w, c := testutil.NewTestContext(http.MethodGet, "/rooms/abc", nil)
		c.Params = gin.Params{{Key: "id", Value: "abc"}}

		h.handler.FindByID(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		h.svc.AssertNotCalled(t, "FindByID", mock.Anything)
	})

	t.Run("failure: not found returns 404", func(t *testing.T) {
		h := newHandlerHarness(t)
		h.svc.On("FindByID", uint(99)).Return(nil, assert.AnError)

		w, c := testutil.NewTestContext(http.MethodGet, "/rooms/99", nil)
		c.Params = gin.Params{{Key: "id", Value: "99"}}

		h.handler.FindByID(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
		h.svc.AssertExpectations(t)
	})
}

func TestHandler_ListUsers(t *testing.T) {
	t.Run("success: returns 200 with mapped user responses", func(t *testing.T) {
		h := newHandlerHarness(t)
		users := []roomuser.RoomUser{{UserID: 1}, {UserID: 2}}
		h.svc.On("ListUsers", uint(4), mock.Anything, mock.Anything, mock.Anything).Return(users, int64(2), nil)

		w, c := testutil.NewTestContext(http.MethodGet, "/rooms/4/users?page=1&pageSize=10", nil)
		c.Params = gin.Params{{Key: "id", Value: "4"}}

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
		h.svc.AssertNotCalled(t, "ListUsers", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("failure: service error returns 500", func(t *testing.T) {
		h := newHandlerHarness(t)
		h.svc.On("ListUsers", uint(4), mock.Anything, mock.Anything, mock.Anything).Return(nil, int64(0), assert.AnError)

		w, c := testutil.NewTestContext(http.MethodGet, "/rooms/4/users?page=1&pageSize=10", nil)
		c.Params = gin.Params{{Key: "id", Value: "4"}}

		h.handler.ListUsers(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		h.svc.AssertExpectations(t)
	})
}

func TestHandler_UpdateUserRole(t *testing.T) {
	t.Run("success: updates role, broadcasts, returns 200", func(t *testing.T) {
		h := newHandlerHarness(t)
		h.svc.On("UpdateUserRole", uint(4), uint(7), role.RoleSpeaker, uint(1)).Return(nil)
		h.hub.On("BroadcastToRoom", uint(4), ws.EventUserRoleUpdated, mock.MatchedBy(func(payload any) bool {
			gh, ok := payload.(gin.H)
			return ok && gh["userId"] == 7 && gh["role"] == role.RoleSpeaker
		})).Return()

		w, c := testutil.NewTestContext(http.MethodPatch, "/rooms/4/users/7/role", UpdateUserRoleParams{Role: role.RoleSpeaker})
		c.Params = gin.Params{{Key: "id", Value: "4"}, {Key: "userId", Value: "7"}}
		c.Set("userId", uint(1))

		h.handler.UpdateUserRole(c)

		assert.Equal(t, http.StatusOK, w.Code)
		h.svc.AssertExpectations(t)
		h.hub.AssertExpectations(t)
	})

	t.Run("failure: forbidden error from service returns 403, no broadcast", func(t *testing.T) {
		h := newHandlerHarness(t)
		h.svc.On("UpdateUserRole", uint(4), uint(7), role.RoleSpeaker, uint(1)).Return(httpx.ErrForbidden)

		w, c := testutil.NewTestContext(http.MethodPatch, "/rooms/4/users/7/role", UpdateUserRoleParams{Role: role.RoleSpeaker})
		c.Params = gin.Params{{Key: "id", Value: "4"}, {Key: "userId", Value: "7"}}
		c.Set("userId", uint(1))

		h.handler.UpdateUserRole(c)

		assert.Equal(t, http.StatusForbidden, w.Code)
		h.svc.AssertExpectations(t)
		h.hub.AssertNotCalled(t, "BroadcastToRoom", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("failure: non-numeric user ID returns 400 before service is called", func(t *testing.T) {
		h := newHandlerHarness(t)

		w, c := testutil.NewTestContext(http.MethodPatch, "/rooms/4/users/abc/role", UpdateUserRoleParams{Role: role.RoleSpeaker})
		c.Params = gin.Params{{Key: "id", Value: "4"}, {Key: "userId", Value: "abc"}}
		c.Set("userId", uint(1))

		h.handler.UpdateUserRole(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		h.svc.AssertNotCalled(t, "UpdateUserRole", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("failure: other service error returns 422", func(t *testing.T) {
		h := newHandlerHarness(t)
		h.svc.On("UpdateUserRole", uint(4), uint(7), role.RoleSpeaker, uint(1)).Return(assert.AnError)

		w, c := testutil.NewTestContext(http.MethodPatch, "/rooms/4/users/7/role", UpdateUserRoleParams{Role: role.RoleSpeaker})
		c.Params = gin.Params{{Key: "id", Value: "4"}, {Key: "userId", Value: "7"}}
		c.Set("userId", uint(1))

		h.handler.UpdateUserRole(c)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		h.svc.AssertExpectations(t)
	})
}

func TestHandler_CurrentRoomUser(t *testing.T) {
	t.Run("success: returns 200 with current room user", func(t *testing.T) {
		h := newHandlerHarness(t)
		want := &roomuser.RoomUser{UserID: 1, RoomID: 4}
		h.svc.On("CurrentRoomUser", uint(4), uint(1)).Return(want, nil)

		w, c := testutil.NewTestContext(http.MethodGet, "/rooms/4/me", nil)
		c.Params = gin.Params{{Key: "id", Value: "4"}}
		c.Set("userId", uint(1))

		h.handler.CurrentRoomUser(c)

		assert.Equal(t, http.StatusOK, w.Code)
		h.svc.AssertExpectations(t)
	})

	t.Run("failure: non-numeric room ID returns 400 before service is called", func(t *testing.T) {
		h := newHandlerHarness(t)

		w, c := testutil.NewTestContext(http.MethodGet, "/rooms/abc/me", nil)
		c.Params = gin.Params{{Key: "id", Value: "abc"}}
		c.Set("userId", uint(1))

		h.handler.CurrentRoomUser(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		h.svc.AssertNotCalled(t, "CurrentRoomUser", mock.Anything, mock.Anything)
	})

	t.Run("failure: nil result (not a member) returns 422", func(t *testing.T) {
		h := newHandlerHarness(t)
		h.svc.On("CurrentRoomUser", uint(4), uint(1)).Return(nil, nil)

		w, c := testutil.NewTestContext(http.MethodGet, "/rooms/4/me", nil)
		c.Params = gin.Params{{Key: "id", Value: "4"}}
		c.Set("userId", uint(1))

		h.handler.CurrentRoomUser(c)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		h.svc.AssertExpectations(t)
	})
}
