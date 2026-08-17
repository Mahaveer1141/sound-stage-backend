package room

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"sound-stage-backend/internal/pkg/listopts"
	"sound-stage-backend/internal/pkg/testutil"
	"sound-stage-backend/internal/role"
	roomuser "sound-stage-backend/internal/room_user"
)

type mockRepository struct{ mock.Mock }

func (m *mockRepository) Create(tx *gorm.DB, input *CreateRoomParams) (*Room, error) {
	args := m.Called(tx, input)
	r, _ := args.Get(0).(*Room)
	return r, args.Error(1)
}
func (m *mockRepository) Update(id uint, input *UpdateRoomParams) (*Room, error) {
	args := m.Called(id, input)
	r, _ := args.Get(0).(*Room)
	return r, args.Error(1)
}
func (m *mockRepository) FindByID(id uint) (*Room, error) {
	args := m.Called(id)
	r, _ := args.Get(0).(*Room)
	return r, args.Error(1)
}
func (m *mockRepository) List(filter RoomFilter, sort listopts.Sort, p listopts.Pagination) ([]Room, error) {
	args := m.Called(filter, sort, p)
	rooms, _ := args.Get(0).([]Room)
	return rooms, args.Error(1)
}
func (m *mockRepository) Count(filter RoomFilter) (int64, error) {
	args := m.Called(filter)
	return args.Get(0).(int64), args.Error(1)
}

type mockRoomUserService struct{ mock.Mock }

func (m *mockRoomUserService) AddUser(userID, roomID uint, roleName role.RoleName) (*roomuser.RoomUser, error) {
	args := m.Called(userID, roomID, roleName)
	ru, _ := args.Get(0).(*roomuser.RoomUser)
	return ru, args.Error(1)
}
func (m *mockRoomUserService) AddUserWithTx(tx *gorm.DB, userID, roomID uint, roleName role.RoleName) (*roomuser.RoomUser, error) {
	args := m.Called(tx, userID, roomID, roleName)
	ru, _ := args.Get(0).(*roomuser.RoomUser)
	return ru, args.Error(1)
}
func (m *mockRoomUserService) RemoveUser(userID, roomID uint) error {
	args := m.Called(userID, roomID)
	return args.Error(0)
}
func (m *mockRoomUserService) ListByRoomID(roomID uint, filter roomuser.RoomUserFilter, sort listopts.Sort, p listopts.Pagination) ([]roomuser.RoomUser, int64, error) {
	args := m.Called(roomID, filter, sort, p)
	rus, _ := args.Get(0).([]roomuser.RoomUser)
	return rus, args.Get(1).(int64), args.Error(2)
}
func (m *mockRoomUserService) FindBy(userID, roomID uint) (*roomuser.RoomUser, error) {
	args := m.Called(userID, roomID)
	ru, _ := args.Get(0).(*roomuser.RoomUser)
	return ru, args.Error(1)
}
func (m *mockRoomUserService) UpdateRole(roomID, userID uint, roleName role.RoleName, actorID uint) error {
	args := m.Called(roomID, userID, roleName, actorID)
	return args.Error(0)
}

type harness struct {
	repo     *mockRepository
	roomUser *mockRoomUserService
	svc      *Service
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	db := testutil.NewIntegrationDB(t)
	repo := new(mockRepository)
	ru := new(mockRoomUserService)
	return &harness{
		repo:     repo,
		roomUser: ru,
		svc:      NewService(repo, ru, db),
	}
}

func TestService_FindByID(t *testing.T) {
	t.Run("success: returns room from repo", func(t *testing.T) {
		h := newHarness(t)
		want := &Room{Name: "Main Stage"}
		h.repo.On("FindByID", uint(1)).Return(want, nil)

		got, err := h.svc.FindByID(1)

		require.NoError(t, err)
		assert.Same(t, want, got)
		h.repo.AssertExpectations(t)
	})

	t.Run("failure: repo error propagated", func(t *testing.T) {
		h := newHarness(t)
		repoErr := errors.New("not found")
		h.repo.On("FindByID", uint(99)).Return(nil, repoErr)

		got, err := h.svc.FindByID(99)

		require.Nil(t, got)
		require.ErrorIs(t, err, repoErr)
		h.repo.AssertExpectations(t)
	})
}

func TestService_Create(t *testing.T) {
	t.Run("success: creates room and adds creator as owner in same tx", func(t *testing.T) {
		h := newHarness(t)
		input := &CreateRoomParams{Name: "New Room", CreatorID: 5}
		created := &Room{Name: "New Room"}
		created.ID = 10

		h.repo.On("Create", mock.AnythingOfType("*gorm.DB"), input).Return(created, nil)
		h.roomUser.On("AddUserWithTx", mock.AnythingOfType("*gorm.DB"), uint(5), uint(10), role.RoleOwner).
			Return(&roomuser.RoomUser{}, nil)

		got, err := h.svc.Create(input)

		require.NoError(t, err)
		assert.Same(t, created, got)
		h.repo.AssertExpectations(t)
		h.roomUser.AssertExpectations(t)
	})

	t.Run("failure: AddUserWithTx error rolls back and room creation is not returned", func(t *testing.T) {
		h := newHarness(t)
		input := &CreateRoomParams{Name: "New Room", CreatorID: 5}
		created := &Room{Name: "New Room"}
		created.ID = 10

		h.repo.On("Create", mock.AnythingOfType("*gorm.DB"), input).Return(created, nil)
		addErr := errors.New("failed to add owner")
		h.roomUser.On("AddUserWithTx", mock.AnythingOfType("*gorm.DB"), uint(5), uint(10), role.RoleOwner).
			Return(nil, addErr)

		got, err := h.svc.Create(input)

		require.Nil(t, got)
		require.ErrorIs(t, err, addErr)
		h.repo.AssertExpectations(t)
		h.roomUser.AssertExpectations(t)
	})
}

func TestService_Update(t *testing.T) {
	t.Run("success: returns updated room from repo", func(t *testing.T) {
		h := newHarness(t)
		input := &UpdateRoomParams{Name: "Renamed"}
		updated := &Room{Name: "Renamed"}
		h.repo.On("Update", uint(3), input).Return(updated, nil)

		got, err := h.svc.Update(3, input)

		require.NoError(t, err)
		assert.Same(t, updated, got)
		h.repo.AssertExpectations(t)
	})

	t.Run("failure: repo error propagated", func(t *testing.T) {
		h := newHarness(t)
		input := &UpdateRoomParams{Name: "Renamed"}
		repoErr := errors.New("room not found")
		h.repo.On("Update", uint(3), input).Return(nil, repoErr)

		got, err := h.svc.Update(3, input)

		require.Nil(t, got)
		require.ErrorIs(t, err, repoErr)
		h.repo.AssertExpectations(t)
	})
}

func TestService_List(t *testing.T) {
	t.Run("success: returns rooms and total count", func(t *testing.T) {
		h := newHarness(t)
		filter := RoomFilter{}
		sort := listopts.Sort{}
		p := listopts.Pagination{Page: 1, PageSize: 10}
		rooms := []Room{{Name: "A"}, {Name: "B"}}

		h.repo.On("List", filter, sort, p).Return(rooms, nil)
		h.repo.On("Count", filter).Return(int64(2), nil)

		got, count, err := h.svc.List(filter, sort, p)

		require.NoError(t, err)
		assert.Equal(t, rooms, got)
		assert.Equal(t, int64(2), count)
		h.repo.AssertExpectations(t)
	})

	t.Run("failure: List error short-circuits before Count is called", func(t *testing.T) {
		h := newHarness(t)
		filter := RoomFilter{}
		sort := listopts.Sort{}
		p := listopts.Pagination{Page: 1, PageSize: 10}
		listErr := errors.New("query failed")

		h.repo.On("List", filter, sort, p).Return(nil, listErr)

		got, count, err := h.svc.List(filter, sort, p)

		require.Nil(t, got)
		require.Zero(t, count)
		require.ErrorIs(t, err, listErr)
		h.repo.AssertNotCalled(t, "Count", mock.Anything)
		h.repo.AssertExpectations(t)
	})

	t.Run("failure: Count error after successful List still fails the call", func(t *testing.T) {
		h := newHarness(t)
		filter := RoomFilter{}
		sort := listopts.Sort{}
		p := listopts.Pagination{Page: 1, PageSize: 10}
		rooms := []Room{{Name: "A"}}
		countErr := errors.New("count query failed")

		h.repo.On("List", filter, sort, p).Return(rooms, nil)
		h.repo.On("Count", filter).Return(int64(0), countErr)

		got, count, err := h.svc.List(filter, sort, p)

		require.Nil(t, got)
		require.Zero(t, count)
		require.ErrorIs(t, err, countErr)
		h.repo.AssertExpectations(t)
	})
}

func TestService_ListUsers(t *testing.T) {
	t.Run("success: delegates to roomUserService", func(t *testing.T) {
		h := newHarness(t)
		filter := roomuser.RoomUserFilter{}
		sort := listopts.Sort{}
		p := listopts.Pagination{Page: 1, PageSize: 10}
		want := []roomuser.RoomUser{{}}

		h.roomUser.On("ListByRoomID", uint(4), filter, sort, p).Return(want, int64(1), nil)

		got, count, err := h.svc.ListUsers(4, filter, sort, p)

		require.NoError(t, err)
		assert.Equal(t, want, got)
		assert.Equal(t, int64(1), count)
		h.roomUser.AssertExpectations(t)
	})

	t.Run("failure: roomUserService error propagated", func(t *testing.T) {
		h := newHarness(t)
		filter := roomuser.RoomUserFilter{}
		sort := listopts.Sort{}
		p := listopts.Pagination{Page: 1, PageSize: 10}
		svcErr := errors.New("query failed")

		h.roomUser.On("ListByRoomID", uint(4), filter, sort, p).Return(nil, int64(0), svcErr)

		got, count, err := h.svc.ListUsers(4, filter, sort, p)

		require.Nil(t, got)
		require.Zero(t, count)
		require.ErrorIs(t, err, svcErr)
		h.roomUser.AssertExpectations(t)
	})
}

func TestService_UpdateUserRole(t *testing.T) {
	t.Run("success: delegates to roomUserService", func(t *testing.T) {
		h := newHarness(t)
		h.roomUser.On("UpdateRole", uint(4), uint(7), role.RoleModerator, uint(1)).Return(nil)

		err := h.svc.UpdateUserRole(4, 7, role.RoleModerator, 1)

		require.NoError(t, err)
		h.roomUser.AssertExpectations(t)
	})

	t.Run("failure: roomUserService error propagated (e.g. unauthorized actor)", func(t *testing.T) {
		h := newHarness(t)
		roleErr := errors.New("actor lacks permission")
		h.roomUser.On("UpdateRole", uint(4), uint(7), role.RoleModerator, uint(1)).Return(roleErr)

		err := h.svc.UpdateUserRole(4, 7, role.RoleModerator, 1)

		require.ErrorIs(t, err, roleErr)
		h.roomUser.AssertExpectations(t)
	})
}

func TestService_CurrentRoomUser(t *testing.T) {
	t.Run("success: delegates to roomUserService.FindBy", func(t *testing.T) {
		h := newHarness(t)
		want := &roomuser.RoomUser{}
		h.roomUser.On("FindBy", uint(7), uint(4)).Return(want, nil)

		got, err := h.svc.CurrentRoomUser(4, 7)

		require.NoError(t, err)
		assert.Same(t, want, got)
		h.roomUser.AssertExpectations(t)
	})

	t.Run("failure: roomUserService error propagated (e.g. not a member)", func(t *testing.T) {
		h := newHarness(t)
		notFoundErr := errors.New("not a member of this room")
		h.roomUser.On("FindBy", uint(7), uint(4)).Return(nil, notFoundErr)

		got, err := h.svc.CurrentRoomUser(4, 7)

		require.Nil(t, got)
		require.ErrorIs(t, err, notFoundErr)
		h.roomUser.AssertExpectations(t)
	})
}
