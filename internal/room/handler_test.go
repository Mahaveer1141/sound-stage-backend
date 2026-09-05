package room

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"sound-stage-backend/internal/pkg/listopts"
	"sound-stage-backend/internal/pkg/testutil"
	"sound-stage-backend/internal/role"
	roomuser "sound-stage-backend/internal/room_user"
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
func (m *mockRoomService) FindByID(id, userID uint) (*Room, error) {
	args := m.Called(id, userID)
	r, _ := args.Get(0).(*Room)
	return r, args.Error(1)
}
func (m *mockRoomService) Create(input *CreateRoomParams) (*Room, error) {
	args := m.Called(input)
	r, _ := args.Get(0).(*Room)
	return r, args.Error(1)
}
func (m *mockRoomService) Update(id, userID uint, input *UpdateRoomParams) (*Room, error) {
	args := m.Called(id, userID, input)
	r, _ := args.Get(0).(*Room)
	return r, args.Error(1)
}
func (m *mockRoomService) ViewerContext(roomID, userID uint) (*RoomViewer, error) {
	args := m.Called(roomID, userID)
	v, _ := args.Get(0).(*RoomViewer)
	return v, args.Error(1)
}
func (m *mockRoomService) ViewerContexts(roomIDs []uint, userID uint) (map[uint]*RoomViewer, error) {
	args := m.Called(roomIDs, userID)
	v, _ := args.Get(0).(map[uint]*RoomViewer)
	return v, args.Error(1)
}
func (m *mockRoomService) UpdatePrivateCode(roomID uint) error {
	args := m.Called(roomID)
	return args.Error(0)
}

type handlerHarness struct {
	svc     *mockRoomService
	handler *Handler
}

func newHandlerHarness(t *testing.T) *handlerHarness {
	t.Helper()
	svc := new(mockRoomService)
	return &handlerHarness{
		svc:     svc,
		handler: NewHandler(svc),
	}
}

func TestHandler_Create(t *testing.T) {
	t.Run("success: creates room and returns 200", func(t *testing.T) {
		h := newHandlerHarness(t)
		created := &Room{Name: "Main Stage"}
		h.svc.On("Create", mock.MatchedBy(func(in *CreateRoomParams) bool {
			return in.Name == "Main Stage" && in.CreatorID == 42
		})).Return(created, nil)
		h.svc.On("ViewerContext", uint(0), uint(42)).Return(&RoomViewer{RoomUser: &roomuser.RoomUser{Role: role.Role{Name: role.RoleOwner}}}, nil)

		w, c := testutil.NewTestContext(http.MethodPost, "/rooms", CreateRoomParams{Name: "Main Stage", Type: RoomTypePublic})
		c.Set("userId", uint(42))

		h.handler.Create(c)

		assert.Equal(t, http.StatusOK, w.Code)
		h.svc.AssertExpectations(t)
	})

	t.Run("failure: service error returns 422", func(t *testing.T) {
		h := newHandlerHarness(t)
		h.svc.On("Create", mock.Anything).Return(nil, assert.AnError)

		w, c := testutil.NewTestContext(http.MethodPost, "/rooms", CreateRoomParams{Name: "Main Stage", Type: RoomTypePublic})
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
		h.svc.On("Update", uint(5), uint(42), mock.AnythingOfType("*room.UpdateRoomParams")).Return(updated, nil)
		h.svc.On("ViewerContext", uint(0), uint(42)).Return(&RoomViewer{RoomUser: &roomuser.RoomUser{Role: role.Role{Name: role.RoleOwner}}}, nil)

		w, c := testutil.NewTestContext(http.MethodPatch, "/rooms/5", UpdateRoomParams{Name: "Renamed", Type: RoomTypePublic})
		c.Params = gin.Params{{Key: "id", Value: "5"}}
		c.Set("userId", uint(42))

		h.handler.Update(c)

		assert.Equal(t, http.StatusOK, w.Code)
		h.svc.AssertExpectations(t)
	})

	t.Run("failure: non-numeric room ID returns 400 before service is called", func(t *testing.T) {
		h := newHandlerHarness(t)

		w, c := testutil.NewTestContext(http.MethodPatch, "/rooms/abc", UpdateRoomParams{Name: "Renamed", Type: RoomTypePublic})
		c.Params = gin.Params{{Key: "id", Value: "abc"}}
		c.Set("userId", uint(42))

		h.handler.Update(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		h.svc.AssertNotCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("failure: service error returns 422", func(t *testing.T) {
		h := newHandlerHarness(t)
		h.svc.On("Update", uint(5), uint(42), mock.Anything).Return(nil, assert.AnError)

		w, c := testutil.NewTestContext(http.MethodPatch, "/rooms/5", UpdateRoomParams{Name: "Renamed", Type: RoomTypePublic})
		c.Params = gin.Params{{Key: "id", Value: "5"}}
		c.Set("userId", uint(42))

		h.handler.Update(c)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		h.svc.AssertExpectations(t)
	})
}

func TestHandler_UpdatePrivateCode(t *testing.T) {
	t.Run("success: updates private code and returns 200", func(t *testing.T) {
		h := newHandlerHarness(t)
		h.svc.On("UpdatePrivateCode", uint(5)).Return(nil)

		w, c := testutil.NewTestContext(http.MethodPatch, "/rooms/5/private-code", nil)
		c.Params = gin.Params{{Key: "id", Value: "5"}}

		h.handler.UpdatePrivateCode(c)

		assert.Equal(t, http.StatusOK, w.Code)
		h.svc.AssertExpectations(t)
	})

	t.Run("failure: non-numeric room ID returns 400 before service is called", func(t *testing.T) {
		h := newHandlerHarness(t)

		w, c := testutil.NewTestContext(http.MethodPatch, "/rooms/abc/private-code", nil)
		c.Params = gin.Params{{Key: "id", Value: "abc"}}

		h.handler.UpdatePrivateCode(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		h.svc.AssertNotCalled(t, "UpdatePrivateCode", mock.Anything)
	})

	t.Run("failure: service error returns 422", func(t *testing.T) {
		h := newHandlerHarness(t)
		h.svc.On("UpdatePrivateCode", uint(5)).Return(assert.AnError)

		w, c := testutil.NewTestContext(http.MethodPatch, "/rooms/5/private-code", nil)
		c.Params = gin.Params{{Key: "id", Value: "5"}}

		h.handler.UpdatePrivateCode(c)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		h.svc.AssertExpectations(t)
	})
}

func TestHandler_List(t *testing.T) {
	t.Run("success: returns 200 with paginated rooms", func(t *testing.T) {
		h := newHandlerHarness(t)
		rooms := []Room{{Name: "A"}, {Name: "B"}}
		h.svc.On("List", mock.Anything, mock.Anything, mock.Anything).Return(rooms, int64(2), nil)
		h.svc.On("ViewerContexts", mock.Anything, mock.Anything).Return(map[uint]*RoomViewer{}, nil)

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
		want.ID = 7
		h.svc.On("FindByID", uint(7), uint(1)).Return(want, nil)
		h.svc.On("ViewerContext", uint(7), uint(1)).Return(&RoomViewer{RoomUser: &roomuser.RoomUser{Role: role.Role{Name: role.RoleListener}}}, nil)

		w, c := testutil.NewTestContext(http.MethodGet, "/rooms/7", nil)
		c.Params = gin.Params{{Key: "id", Value: "7"}}
		c.Set("userId", uint(1))

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
		h.svc.On("FindByID", uint(99), uint(0)).Return(nil, assert.AnError)

		w, c := testutil.NewTestContext(http.MethodGet, "/rooms/99", nil)
		c.Params = gin.Params{{Key: "id", Value: "99"}}

		h.handler.FindByID(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
		h.svc.AssertExpectations(t)
	})
}
