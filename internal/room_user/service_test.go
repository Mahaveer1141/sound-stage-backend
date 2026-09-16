package roomuser

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"sound-stage-backend/internal/model"
	"sound-stage-backend/internal/pkg/httpx"
	"sound-stage-backend/internal/pkg/listopts"
	"sound-stage-backend/internal/role"
	roomstate "sound-stage-backend/internal/room_state"
)

type mockRepo struct{ mock.Mock }

func (m *mockRepo) Create(tx *gorm.DB, userID, roomID, roleID uint) (*RoomUser, error) {
	args := m.Called(tx, userID, roomID, roleID)
	ru, _ := args.Get(0).(*RoomUser)
	return ru, args.Error(1)
}
func (m *mockRepo) FindBy(userID, roomID uint) (*RoomUser, error) {
	args := m.Called(userID, roomID)
	ru, _ := args.Get(0).(*RoomUser)
	return ru, args.Error(1)
}
func (m *mockRepo) FindAnyBy(userID, roomID uint) (*RoomUser, error) {
	args := m.Called(userID, roomID)
	ru, _ := args.Get(0).(*RoomUser)
	return ru, args.Error(1)
}
func (m *mockRepo) UpdateActivity(ru *RoomUser, activity Activity) error {
	args := m.Called(ru, activity)
	return args.Error(0)
}
func (m *mockRepo) HasRoles(userID, roomID uint, permissions []role.RoleName) (bool, error) {
	args := m.Called(userID, roomID, permissions)
	return args.Bool(0), args.Error(1)
}
func (m *mockRepo) ListByRoomID(roomID uint, filter RoomUserFilter, sort listopts.Sort, c listopts.Cursor) ([]RoomUser, bool, string, error) {
	args := m.Called(roomID, filter, sort, c)
	rus, _ := args.Get(0).([]RoomUser)
	return rus, args.Bool(1), args.Get(2).(string), args.Error(3)
}
func (m *mockRepo) ListByUserIDs(roomID uint, userIDs []uint) ([]RoomUser, error) {
	args := m.Called(roomID, userIDs)
	rus, _ := args.Get(0).([]RoomUser)
	return rus, args.Error(1)
}
func (m *mockRepo) CountByRoomID(roomID uint, filter RoomUserFilter) (int64, error) {
	args := m.Called(roomID, filter)
	return args.Get(0).(int64), args.Error(1)
}
func (m *mockRepo) CountByRoomIDs(roomIDs []uint, filter RoomUserFilter) (map[uint]int64, error) {
	args := m.Called(roomIDs, filter)
	res, _ := args.Get(0).(map[uint]int64)
	return res, args.Error(1)
}
func (m *mockRepo) UpdateRole(roomID, userID, roleID uint) error {
	args := m.Called(roomID, userID, roleID)
	return args.Error(0)
}
func (m *mockRepo) Delete(roomID, userID uint) error {
	args := m.Called(roomID, userID)
	return args.Error(0)
}
func (m *mockRepo) MapByUserAndRoomIDs(userID uint, roomIDs []uint) (map[uint]*RoomUser, error) {
	args := m.Called(userID, roomIDs)
	res, _ := args.Get(0).(map[uint]*RoomUser)
	return res, args.Error(1)
}
func (m *mockRepo) IsBlocked(roomID, userID uint) (bool, error) {
	args := m.Called(roomID, userID)
	return args.Bool(0), args.Error(1)
}
func (m *mockRepo) Block(roomID, userID, blockedByID, listenerRoleID uint) error {
	args := m.Called(roomID, userID, blockedByID, listenerRoleID)
	return args.Error(0)
}
func (m *mockRepo) Unblock(roomID, userID uint) error {
	args := m.Called(roomID, userID)
	return args.Error(0)
}
func (m *mockRepo) ListBlockedByRoomID(roomID uint, filter RoomUserFilter, p listopts.Pagination) ([]RoomUser, error) {
	args := m.Called(roomID, filter, p)
	rus, _ := args.Get(0).([]RoomUser)
	return rus, args.Error(1)
}
func (m *mockRepo) CountBlockedByRoomID(roomID uint, filter RoomUserFilter) (int64, error) {
	args := m.Called(roomID, filter)
	return args.Get(0).(int64), args.Error(1)
}

type mockRoleFinder struct{ mock.Mock }

func (m *mockRoleFinder) FindByName(name role.RoleName) (*role.Role, error) {
	args := m.Called(name)
	r, _ := args.Get(0).(*role.Role)
	return r, args.Error(1)
}

type mockRevoker struct{ mock.Mock }

func (m *mockRevoker) RevokePublishing(roomID, userID uint) {
	m.Called(roomID, userID)
}

type mockStateReader struct{ mock.Mock }

func (m *mockStateReader) GetParticipantStates(ctx context.Context, roomID uint, userIDs []uint) (map[uint]*roomstate.ParticipantState, error) {
	args := m.Called(ctx, roomID, userIDs)
	res, _ := args.Get(0).(map[uint]*roomstate.ParticipantState)
	return res, args.Error(1)
}
func (m *mockStateReader) GetRaisedHands(ctx context.Context, roomID uint, p listopts.Pagination) ([]uint, error) {
	args := m.Called(ctx, roomID, p)
	ids, _ := args.Get(0).([]uint)
	return ids, args.Error(1)
}
func (m *mockStateReader) CountRaisedHands(ctx context.Context, roomID uint) (int64, error) {
	args := m.Called(ctx, roomID)
	return args.Get(0).(int64), args.Error(1)
}
func (m *mockStateReader) SetMuted(ctx context.Context, roomID, userID uint, isMuted bool) error {
	args := m.Called(ctx, roomID, userID, isMuted)
	return args.Error(0)
}
func (m *mockStateReader) SetHandRaised(ctx context.Context, roomID, userID uint, isHandRaised bool) error {
	args := m.Called(ctx, roomID, userID, isHandRaised)
	return args.Error(0)
}
func (m *mockStateReader) Leave(ctx context.Context, roomID, userID uint) error {
	args := m.Called(ctx, roomID, userID)
	return args.Error(0)
}

type harness struct {
	repo    *mockRepo
	roles   *mockRoleFinder
	revoker *mockRevoker
	state   *mockStateReader
	svc     *Service
}

func newHarness() *harness {
	repo := new(mockRepo)
	roles := new(mockRoleFinder)
	revoker := new(mockRevoker)
	state := new(mockStateReader)
	return &harness{
		repo:    repo,
		roles:   roles,
		revoker: revoker,
		state:   state,
		svc:     NewService(repo, roles, revoker, state, NewAuthz(repo)),
	}
}

func (h *harness) assertAllExpectations(t *testing.T) {
	t.Helper()
	h.repo.AssertExpectations(t)
	h.roles.AssertExpectations(t)
	h.revoker.AssertExpectations(t)
	h.state.AssertExpectations(t)
}

func TestService_Rejoin(t *testing.T) {
	t.Run("success: marks existing member as joined", func(t *testing.T) {
		h := newHarness()
		existing := &RoomUser{UserID: 1, RoomID: 2}
		h.repo.On("UpdateActivity", existing, ActivityJoin).Return(nil)

		err := h.svc.Rejoin(existing)

		require.NoError(t, err)
		h.assertAllExpectations(t)
	})

	t.Run("failure: UpdateActivity error is propagated", func(t *testing.T) {
		h := newHarness()
		existing := &RoomUser{UserID: 1, RoomID: 2}
		updateErr := errors.New("update failed")
		h.repo.On("UpdateActivity", existing, ActivityJoin).Return(updateErr)

		err := h.svc.Rejoin(existing)

		require.ErrorIs(t, err, updateErr)
		h.assertAllExpectations(t)
	})
}

func TestService_Create(t *testing.T) {
	t.Run("success: creates with explicit role", func(t *testing.T) {
		h := newHarness()
		h.roles.On("FindByName", role.RoleListener).Return(&role.Role{BaseModel: model.BaseModel{ID: 4}, Name: role.RoleListener}, nil)
		created := &RoomUser{UserID: 1, RoomID: 2}
		h.repo.On("Create", (*gorm.DB)(nil), uint(1), uint(2), uint(4)).Return(created, nil)

		got, err := h.svc.Create(nil, 1, 2, role.RoleListener)

		require.NoError(t, err)
		assert.Same(t, created, got)
		h.assertAllExpectations(t)
	})

	t.Run("success: empty role name defaults to RoleListener", func(t *testing.T) {
		h := newHarness()
		h.roles.On("FindByName", role.RoleListener).Return(&role.Role{BaseModel: model.BaseModel{ID: 4}, Name: role.RoleListener}, nil)
		created := &RoomUser{UserID: 1, RoomID: 2}
		h.repo.On("Create", (*gorm.DB)(nil), uint(1), uint(2), uint(4)).Return(created, nil)

		got, err := h.svc.Create(nil, 1, 2, role.RoleName(""))

		require.NoError(t, err)
		assert.Same(t, created, got)
		h.assertAllExpectations(t)
	})

	t.Run("failure: role lookup error is propagated, Create never called", func(t *testing.T) {
		h := newHarness()
		roleErr := errors.New("unknown role")
		h.roles.On("FindByName", role.RoleName("no-role")).Return(nil, roleErr)

		got, err := h.svc.Create(nil, 1, 2, role.RoleName("no-role"))

		require.Nil(t, got)
		require.ErrorIs(t, err, roleErr)
		h.repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: Create error is propagated", func(t *testing.T) {
		h := newHarness()
		createErr := errors.New("insert failed")
		h.roles.On("FindByName", role.RoleListener).Return(&role.Role{BaseModel: model.BaseModel{ID: 4}, Name: role.RoleListener}, nil)
		h.repo.On("Create", (*gorm.DB)(nil), uint(1), uint(2), uint(4)).Return(nil, createErr)

		got, err := h.svc.Create(nil, 1, 2, role.RoleListener)

		require.Nil(t, got)
		require.ErrorIs(t, err, createErr)
		h.assertAllExpectations(t)
	})
}

func TestService_FindBy(t *testing.T) {
	t.Run("success: returns record from repo", func(t *testing.T) {
		h := newHarness()
		want := &RoomUser{UserID: 1, RoomID: 2}
		h.repo.On("FindBy", uint(1), uint(2)).Return(want, nil)

		got, err := h.svc.FindBy(1, 2)

		require.NoError(t, err)
		assert.Same(t, want, got)
		h.assertAllExpectations(t)
	})

	t.Run("failure: repo error propagated", func(t *testing.T) {
		h := newHarness()
		repoErr := errors.New("db down")
		h.repo.On("FindBy", uint(1), uint(2)).Return(nil, repoErr)

		got, err := h.svc.FindBy(1, 2)

		require.Nil(t, got)
		require.ErrorIs(t, err, repoErr)
		h.assertAllExpectations(t)
	})
}

func TestService_MapByUserAndRoomIDs(t *testing.T) {
	t.Run("success: returns map from repo", func(t *testing.T) {
		h := newHarness()
		want := map[uint]*RoomUser{2: {UserID: 1, RoomID: 2}}
		h.repo.On("MapByUserAndRoomIDs", uint(1), []uint{2, 3}).Return(want, nil)

		got, err := h.svc.MapByUserAndRoomIDs(1, []uint{2, 3})

		require.NoError(t, err)
		assert.Equal(t, want, got)
		h.assertAllExpectations(t)
	})

	t.Run("failure: repo error propagated", func(t *testing.T) {
		h := newHarness()
		repoErr := errors.New("db down")
		h.repo.On("MapByUserAndRoomIDs", uint(1), []uint{2, 3}).Return(nil, repoErr)

		got, err := h.svc.MapByUserAndRoomIDs(1, []uint{2, 3})

		require.Nil(t, got)
		require.ErrorIs(t, err, repoErr)
		h.assertAllExpectations(t)
	})
}

func TestService_RemoveUser(t *testing.T) {
	t.Run("success: marks activity as leave and clears participant state", func(t *testing.T) {
		h := newHarness()
		existing := &RoomUser{UserID: 1, RoomID: 2}
		h.repo.On("FindBy", uint(1), uint(2)).Return(existing, nil)
		h.repo.On("UpdateActivity", existing, ActivityLeave).Return(nil)
		h.state.On("Leave", mock.Anything, uint(2), uint(1)).Return(nil)

		got, err := h.svc.RemoveUser(context.Background(), 1, 2)

		require.NoError(t, err)
		assert.Same(t, existing, got)
		h.assertAllExpectations(t)
	})

	t.Run("failure: member not found returns ErrRecordNotFound", func(t *testing.T) {
		h := newHarness()
		h.repo.On("FindBy", uint(1), uint(2)).Return(nil, nil)

		got, err := h.svc.RemoveUser(context.Background(), 1, 2)

		require.Nil(t, got)
		require.ErrorIs(t, err, httpx.ErrRecordNotFound)
		h.repo.AssertNotCalled(t, "UpdateActivity", mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: FindBy error is propagated", func(t *testing.T) {
		h := newHarness()
		findErr := errors.New("db down")
		h.repo.On("FindBy", uint(1), uint(2)).Return(nil, findErr)

		got, err := h.svc.RemoveUser(context.Background(), 1, 2)

		require.Nil(t, got)
		require.ErrorIs(t, err, findErr)
		h.assertAllExpectations(t)
	})

	t.Run("failure: UpdateActivity error is propagated", func(t *testing.T) {
		h := newHarness()
		existing := &RoomUser{UserID: 1, RoomID: 2}
		updateErr := errors.New("update failed")
		h.repo.On("FindBy", uint(1), uint(2)).Return(existing, nil)
		h.repo.On("UpdateActivity", existing, ActivityLeave).Return(updateErr)

		got, err := h.svc.RemoveUser(context.Background(), 1, 2)

		require.Nil(t, got)
		require.ErrorIs(t, err, updateErr)
		h.state.AssertNotCalled(t, "Leave", mock.Anything, mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})
}

func TestService_ListByRoomID(t *testing.T) {
	t.Run("success: returns users and total count", func(t *testing.T) {
		h := newHarness()
		filter := RoomUserFilter{}
		sort := listopts.Sort{Field: "last_joined_at", Order: "asc"}
		cur := listopts.Cursor{Cursor: "", Limit: 10}
		users := []RoomUser{{UserID: 1}, {UserID: 2}}

		h.repo.On("FindAnyBy", uint(1), uint(4)).Return(&RoomUser{IsBlocked: false}, nil)
		h.repo.On("ListByRoomID", uint(4), filter, sort, cur).Return(users, false, "", nil)
		h.repo.On("CountByRoomID", uint(4), filter).Return(int64(2), nil)
		h.state.On("GetParticipantStates", mock.Anything, uint(4), []uint{1, 2}).
			Return(map[uint]*roomstate.ParticipantState{
				1: {UserID: 1, IsMuted: true},
				2: {UserID: 2, IsHandRaised: true},
			}, nil)

		got, count, _, _, err := h.svc.ListByRoomID(context.Background(), 4, 1, filter, cur)

		require.NoError(t, err)
		assert.Equal(t, users, got)
		assert.True(t, got[0].IsMuted)
		assert.False(t, got[0].IsHandRaised)
		assert.False(t, got[1].IsMuted)
		assert.True(t, got[1].IsHandRaised)
		assert.Equal(t, int64(2), count)
		h.assertAllExpectations(t)
	})

	t.Run("failure: state fetch error fails the call", func(t *testing.T) {
		h := newHarness()
		filter := RoomUserFilter{}
		sort := listopts.Sort{Field: "last_joined_at", Order: "asc"}
		cur := listopts.Cursor{Cursor: "", Limit: 10}
		users := []RoomUser{{UserID: 1}}
		stateErr := errors.New("redis down")

		h.repo.On("FindAnyBy", uint(1), uint(4)).Return(&RoomUser{IsBlocked: false}, nil)
		h.repo.On("ListByRoomID", uint(4), filter, sort, cur).Return(users, false, "", nil)
		h.repo.On("CountByRoomID", uint(4), filter).Return(int64(1), nil)
		h.state.On("GetParticipantStates", mock.Anything, uint(4), []uint{1}).Return(nil, stateErr)

		got, count, _, _, err := h.svc.ListByRoomID(context.Background(), 4, 1, filter, cur)

		require.Nil(t, got)
		require.Zero(t, count)
		require.ErrorIs(t, err, stateErr)
		h.assertAllExpectations(t)
	})

	t.Run("failure: List error short-circuits before Count", func(t *testing.T) {
		h := newHarness()
		filter := RoomUserFilter{}
		sort := listopts.Sort{Field: "last_joined_at", Order: "asc"}
		cur := listopts.Cursor{Cursor: "", Limit: 10}
		listErr := errors.New("query failed")
		h.repo.On("FindAnyBy", uint(1), uint(4)).Return(&RoomUser{IsBlocked: false}, nil)
		h.repo.On("ListByRoomID", uint(4), filter, sort, cur).Return(nil, false, "", listErr)

		got, count, _, _, err := h.svc.ListByRoomID(context.Background(), 4, 1, filter, cur)

		require.Nil(t, got)
		require.Zero(t, count)
		require.ErrorIs(t, err, listErr)
		h.repo.AssertNotCalled(t, "CountByRoomID", mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: Count error after successful List still fails the call", func(t *testing.T) {
		h := newHarness()
		filter := RoomUserFilter{}
		sort := listopts.Sort{Field: "last_joined_at", Order: "asc"}
		cur := listopts.Cursor{Cursor: "", Limit: 10}
		users := []RoomUser{{UserID: 1}}
		countErr := errors.New("count failed")
		h.repo.On("FindAnyBy", uint(1), uint(4)).Return(&RoomUser{IsBlocked: false}, nil)
		h.repo.On("ListByRoomID", uint(4), filter, sort, cur).Return(users, false, "", nil)
		h.repo.On("CountByRoomID", uint(4), filter).Return(int64(0), countErr)

		got, count, _, _, err := h.svc.ListByRoomID(context.Background(), 4, 1, filter, cur)

		require.Nil(t, got)
		require.Zero(t, count)
		require.ErrorIs(t, err, countErr)
		h.assertAllExpectations(t)
	})

	t.Run("failure: non-member gets ErrForbidden before repo is called", func(t *testing.T) {
		h := newHarness()
		filter := RoomUserFilter{}
		cur := listopts.Cursor{Cursor: "", Limit: 10}
		h.repo.On("FindAnyBy", uint(1), uint(4)).Return(nil, nil)

		got, count, _, _, err := h.svc.ListByRoomID(context.Background(), 4, 1, filter, cur)

		require.Nil(t, got)
		require.Zero(t, count)
		require.ErrorIs(t, err, httpx.ErrForbidden)
		h.repo.AssertNotCalled(t, "ListByRoomID", mock.Anything, mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: blocked user gets ErrUserBlocked before repo is called", func(t *testing.T) {
		h := newHarness()
		filter := RoomUserFilter{}
		cur := listopts.Cursor{Cursor: "", Limit: 10}
		h.repo.On("FindAnyBy", uint(1), uint(4)).Return(&RoomUser{IsBlocked: true}, nil)

		got, count, _, _, err := h.svc.ListByRoomID(context.Background(), 4, 1, filter, cur)

		require.Nil(t, got)
		require.Zero(t, count)
		require.ErrorIs(t, err, httpx.ErrUserBlocked)
		h.repo.AssertNotCalled(t, "ListByRoomID", mock.Anything, mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: FindAnyBy error is propagated", func(t *testing.T) {
		h := newHarness()
		filter := RoomUserFilter{}
		cur := listopts.Cursor{Cursor: "", Limit: 10}
		findErr := errors.New("find failed")
		h.repo.On("FindAnyBy", uint(1), uint(4)).Return(nil, findErr)

		got, count, _, _, err := h.svc.ListByRoomID(context.Background(), 4, 1, filter, cur)

		require.Nil(t, got)
		require.Zero(t, count)
		require.ErrorIs(t, err, findErr)
		h.repo.AssertNotCalled(t, "ListByRoomID", mock.Anything, mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})
}

func TestService_UpdateRole(t *testing.T) {
	t.Run("success: promoting to non-listener role does not revoke publishing", func(t *testing.T) {
		h := newHarness()
		h.repo.On("HasRoles", uint(1), uint(2), role.RoleAssignmentPermissions[role.RoleSpeaker]).Return(true, nil)
		h.roles.On("FindByName", role.RoleSpeaker).Return(&role.Role{BaseModel: model.BaseModel{ID: 9}, Name: role.RoleSpeaker}, nil)
		h.repo.On("UpdateRole", uint(2), uint(3), uint(9)).Return(nil)
		h.repo.On("FindBy", uint(3), uint(2)).Return(&RoomUser{UserID: 3, RoomID: 2, Role: role.Role{Name: role.RoleSpeaker}}, nil)
		h.state.On("GetParticipantStates", mock.Anything, uint(2), []uint{3}).
			Return(map[uint]*roomstate.ParticipantState{3: {UserID: 3, IsMuted: true}}, nil)

		ru, err := h.svc.UpdateRole(context.Background(), 2, 3, role.RoleSpeaker, 1)

		require.NoError(t, err)
		require.Equal(t, role.RoleSpeaker, ru.Role.Name)
		require.True(t, ru.IsMuted)
		h.revoker.AssertNotCalled(t, "RevokePublishing", mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("success: demoting to listener revokes publishing", func(t *testing.T) {
		h := newHarness()
		h.repo.On("HasRoles", uint(1), uint(2), role.RoleAssignmentPermissions[role.RoleListener]).Return(true, nil)
		h.roles.On("FindByName", role.RoleListener).Return(&role.Role{BaseModel: model.BaseModel{ID: 3}, Name: role.RoleListener}, nil)
		h.repo.On("UpdateRole", uint(2), uint(3), uint(3)).Return(nil)
		h.revoker.On("RevokePublishing", uint(2), uint(3)).Return()
		h.repo.On("FindBy", uint(3), uint(2)).Return(&RoomUser{UserID: 3, RoomID: 2, Role: role.Role{Name: role.RoleListener}}, nil)
		h.state.On("GetParticipantStates", mock.Anything, uint(2), []uint{3}).
			Return(map[uint]*roomstate.ParticipantState{}, nil)

		ru, err := h.svc.UpdateRole(context.Background(), 2, 3, role.RoleListener, 1)

		require.NoError(t, err)
		require.Equal(t, role.RoleListener, ru.Role.Name)
		h.assertAllExpectations(t)
	})

	t.Run("failure: actor without permission gets ErrForbidden, nothing else runs", func(t *testing.T) {
		h := newHarness()
		h.repo.On("HasRoles", uint(1), uint(2), role.RoleAssignmentPermissions[role.RoleSpeaker]).Return(false, nil)

		ru, err := h.svc.UpdateRole(context.Background(), 2, 3, role.RoleSpeaker, 1)

		require.Nil(t, ru)
		require.ErrorIs(t, err, httpx.ErrForbidden)
		h.roles.AssertNotCalled(t, "FindByName", mock.Anything)
		h.repo.AssertNotCalled(t, "UpdateRole", mock.Anything, mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: HasRoles error is propagated", func(t *testing.T) {
		h := newHarness()
		permErr := errors.New("permission check failed")
		h.repo.On("HasRoles", uint(1), uint(2), role.RoleAssignmentPermissions[role.RoleSpeaker]).Return(false, permErr)

		_, err := h.svc.UpdateRole(context.Background(), 2, 3, role.RoleSpeaker, 1)

		require.ErrorIs(t, err, permErr)
		h.assertAllExpectations(t)
	})

	t.Run("failure: role lookup error is propagated, UpdateRole never called", func(t *testing.T) {
		h := newHarness()
		h.repo.On("HasRoles", uint(1), uint(2), role.RoleAssignmentPermissions[role.RoleSpeaker]).Return(true, nil)
		roleErr := errors.New("unknown role")
		h.roles.On("FindByName", role.RoleSpeaker).Return(nil, roleErr)

		_, err := h.svc.UpdateRole(context.Background(), 2, 3, role.RoleSpeaker, 1)

		require.ErrorIs(t, err, roleErr)
		h.repo.AssertNotCalled(t, "UpdateRole", mock.Anything, mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: user not found returns ErrRecordNotFound", func(t *testing.T) {
		h := newHarness()
		h.repo.On("HasRoles", uint(1), uint(2), role.RoleAssignmentPermissions[role.RoleSpeaker]).Return(true, nil)
		h.roles.On("FindByName", role.RoleSpeaker).Return(&role.Role{BaseModel: model.BaseModel{ID: 9}, Name: role.RoleSpeaker}, nil)
		h.repo.On("UpdateRole", uint(2), uint(3), uint(9)).Return(nil)
		h.repo.On("FindBy", uint(3), uint(2)).Return(nil, nil)

		ru, err := h.svc.UpdateRole(context.Background(), 2, 3, role.RoleSpeaker, 1)

		require.Nil(t, ru)
		require.ErrorIs(t, err, httpx.ErrRecordNotFound)
		h.assertAllExpectations(t)
	})

	t.Run("failure: UpdateRole error is propagated, revoker never called", func(t *testing.T) {
		h := newHarness()
		h.repo.On("HasRoles", uint(1), uint(2), role.RoleAssignmentPermissions[role.RoleListener]).Return(true, nil)
		h.roles.On("FindByName", role.RoleListener).Return(&role.Role{BaseModel: model.BaseModel{ID: 3}, Name: role.RoleListener}, nil)
		updateErr := errors.New("update failed")
		h.repo.On("UpdateRole", uint(2), uint(3), uint(3)).Return(updateErr)

		_, err := h.svc.UpdateRole(context.Background(), 2, 3, role.RoleListener, 1)

		require.ErrorIs(t, err, updateErr)
		h.revoker.AssertNotCalled(t, "RevokePublishing", mock.Anything, mock.Anything)
		h.repo.AssertNotCalled(t, "FindBy", mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})
}

func TestService_DeleteUser(t *testing.T) {
	t.Run("success: owner deletes an admin", func(t *testing.T) {
		h := newHarness()
		target := &RoomUser{UserID: 3, Role: role.Role{Name: role.RoleAdmin}}
		h.repo.On("FindBy", uint(1), uint(2)).Return(&RoomUser{Role: role.Role{Name: role.RoleOwner}}, nil)
		h.repo.On("FindAnyBy", uint(3), uint(2)).Return(target, nil)
		h.repo.On("Delete", uint(2), uint(3)).Return(nil)
		h.state.On("Leave", mock.Anything, uint(2), uint(3)).Return(nil)

		got, err := h.svc.DeleteUser(context.Background(), 2, 3, 1)

		require.NoError(t, err)
		assert.Same(t, target, got)
		h.assertAllExpectations(t)
	})

	t.Run("success: non-owner can delete themselves", func(t *testing.T) {
		h := newHarness()
		self := &RoomUser{UserID: 1, Role: role.Role{Name: role.RoleAdmin}}
		h.repo.On("FindBy", uint(1), uint(2)).Return(self, nil)
		h.repo.On("Delete", uint(2), uint(1)).Return(nil)
		h.state.On("Leave", mock.Anything, uint(2), uint(1)).Return(nil)

		got, err := h.svc.DeleteUser(context.Background(), 2, 1, 1)

		require.NoError(t, err)
		assert.Same(t, self, got)
		h.assertAllExpectations(t)
	})

	t.Run("failure: owner cannot delete themselves", func(t *testing.T) {
		h := newHarness()
		h.repo.On("FindBy", uint(1), uint(2)).Return(&RoomUser{Role: role.Role{Name: role.RoleOwner}}, nil)

		got, err := h.svc.DeleteUser(context.Background(), 2, 1, 1)

		require.Nil(t, got)
		require.ErrorIs(t, err, httpx.ErrForbidden)
		h.repo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
		h.state.AssertNotCalled(t, "Leave", mock.Anything, mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: admin cannot delete an owner", func(t *testing.T) {
		h := newHarness()
		h.repo.On("FindBy", uint(1), uint(2)).Return(&RoomUser{Role: role.Role{Name: role.RoleAdmin}}, nil)
		h.repo.On("FindAnyBy", uint(3), uint(2)).Return(&RoomUser{Role: role.Role{Name: role.RoleOwner}}, nil)

		got, err := h.svc.DeleteUser(context.Background(), 2, 3, 1)

		require.Nil(t, got)
		require.ErrorIs(t, err, httpx.ErrForbidden)
		h.repo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: speaker cannot delete a listener", func(t *testing.T) {
		h := newHarness()
		h.repo.On("FindBy", uint(1), uint(2)).Return(&RoomUser{Role: role.Role{Name: role.RoleSpeaker}}, nil)
		h.repo.On("FindAnyBy", uint(3), uint(2)).Return(&RoomUser{Role: role.Role{Name: role.RoleListener}}, nil)

		got, err := h.svc.DeleteUser(context.Background(), 2, 3, 1)

		require.Nil(t, got)
		require.ErrorIs(t, err, httpx.ErrForbidden)
		h.repo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: actor not in room is forbidden", func(t *testing.T) {
		h := newHarness()
		h.repo.On("FindBy", uint(1), uint(2)).Return(nil, nil)

		got, err := h.svc.DeleteUser(context.Background(), 2, 3, 1)

		require.Nil(t, got)
		require.ErrorIs(t, err, httpx.ErrForbidden)
		h.repo.AssertNotCalled(t, "FindAnyBy", uint(3), uint(2))
		h.repo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: target not in room returns ErrRecordNotFound", func(t *testing.T) {
		h := newHarness()
		h.repo.On("FindBy", uint(1), uint(2)).Return(&RoomUser{Role: role.Role{Name: role.RoleOwner}}, nil)
		h.repo.On("FindAnyBy", uint(3), uint(2)).Return(nil, nil)

		got, err := h.svc.DeleteUser(context.Background(), 2, 3, 1)

		require.Nil(t, got)
		require.ErrorIs(t, err, httpx.ErrRecordNotFound)
		h.repo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: FindBy error is propagated", func(t *testing.T) {
		h := newHarness()
		findErr := errors.New("db down")
		h.repo.On("FindBy", uint(1), uint(2)).Return(nil, findErr)

		got, err := h.svc.DeleteUser(context.Background(), 2, 3, 1)

		require.Nil(t, got)
		require.ErrorIs(t, err, findErr)
		h.repo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: Delete error is propagated", func(t *testing.T) {
		h := newHarness()
		deleteErr := errors.New("delete failed")
		h.repo.On("FindBy", uint(1), uint(2)).Return(&RoomUser{Role: role.Role{Name: role.RoleOwner}}, nil)
		h.repo.On("FindAnyBy", uint(3), uint(2)).Return(&RoomUser{Role: role.Role{Name: role.RoleListener}}, nil)
		h.repo.On("Delete", uint(2), uint(3)).Return(deleteErr)

		got, err := h.svc.DeleteUser(context.Background(), 2, 3, 1)

		require.Nil(t, got)
		require.ErrorIs(t, err, deleteErr)
		h.assertAllExpectations(t)
	})

	t.Run("failure: blocked user cannot be deleted", func(t *testing.T) {
		h := newHarness()
		h.repo.On("FindBy", uint(1), uint(2)).Return(&RoomUser{Role: role.Role{Name: role.RoleOwner}}, nil)
		h.repo.On("FindAnyBy", uint(3), uint(2)).Return(&RoomUser{Role: role.Role{Name: role.RoleListener}, IsBlocked: true}, nil)

		got, err := h.svc.DeleteUser(context.Background(), 2, 3, 1)

		require.Nil(t, got)
		require.ErrorIs(t, err, httpx.ErrUserBlocked)
		h.repo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})
}

func TestService_SetMuted(t *testing.T) {
	t.Run("success: user mutes themselves", func(t *testing.T) {
		h := newHarness()
		h.repo.On("FindAnyBy", uint(1), uint(2)).Return(&RoomUser{IsBlocked: false}, nil)
		h.state.On("SetMuted", mock.Anything, uint(2), uint(1), true).Return(nil)

		err := h.svc.SetMuted(context.Background(), 2, 1, 1, true)

		require.NoError(t, err)
		h.assertAllExpectations(t)
	})

	t.Run("success: user unmutes themselves", func(t *testing.T) {
		h := newHarness()
		h.repo.On("FindAnyBy", uint(1), uint(2)).Return(&RoomUser{IsBlocked: false}, nil)
		h.state.On("SetMuted", mock.Anything, uint(2), uint(1), false).Return(nil)

		err := h.svc.SetMuted(context.Background(), 2, 1, 1, false)

		require.NoError(t, err)
		h.assertAllExpectations(t)
	})

	t.Run("success: moderator mutes a listener", func(t *testing.T) {
		h := newHarness()
		h.repo.On("FindBy", uint(1), uint(2)).Return(&RoomUser{Role: role.Role{Name: role.RoleModerator}}, nil)
		h.repo.On("FindAnyBy", uint(3), uint(2)).Return(&RoomUser{Role: role.Role{Name: role.RoleListener}}, nil)
		h.state.On("SetMuted", mock.Anything, uint(2), uint(3), true).Return(nil)

		err := h.svc.SetMuted(context.Background(), 2, 3, 1, true)

		require.NoError(t, err)
		h.assertAllExpectations(t)
	})

	t.Run("failure: blocked user cannot mute", func(t *testing.T) {
		h := newHarness()
		h.repo.On("FindAnyBy", uint(1), uint(2)).Return(&RoomUser{IsBlocked: true}, nil)

		err := h.svc.SetMuted(context.Background(), 2, 1, 1, true)

		require.ErrorIs(t, err, httpx.ErrUserBlocked)
		h.state.AssertNotCalled(t, "SetMuted", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: admin cannot unmute another user", func(t *testing.T) {
		h := newHarness()

		err := h.svc.SetMuted(context.Background(), 2, 3, 1, false)

		require.ErrorIs(t, err, httpx.ErrForbidden)
		h.state.AssertNotCalled(t, "SetMuted", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: speaker cannot mute a listener", func(t *testing.T) {
		h := newHarness()
		h.repo.On("FindBy", uint(1), uint(2)).Return(&RoomUser{Role: role.Role{Name: role.RoleSpeaker}}, nil)
		h.repo.On("FindAnyBy", uint(3), uint(2)).Return(&RoomUser{Role: role.Role{Name: role.RoleListener}}, nil)

		err := h.svc.SetMuted(context.Background(), 2, 3, 1, true)

		require.ErrorIs(t, err, httpx.ErrForbidden)
		h.state.AssertNotCalled(t, "SetMuted", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: actor not in room is forbidden", func(t *testing.T) {
		h := newHarness()
		h.repo.On("FindBy", uint(1), uint(2)).Return(nil, nil)

		err := h.svc.SetMuted(context.Background(), 2, 3, 1, true)

		require.ErrorIs(t, err, httpx.ErrForbidden)
		h.repo.AssertNotCalled(t, "FindAnyBy", mock.Anything, mock.Anything)
		h.state.AssertNotCalled(t, "SetMuted", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: target not in room returns ErrRecordNotFound", func(t *testing.T) {
		h := newHarness()
		h.repo.On("FindBy", uint(1), uint(2)).Return(&RoomUser{Role: role.Role{Name: role.RoleOwner}}, nil)
		h.repo.On("FindAnyBy", uint(3), uint(2)).Return(nil, nil)

		err := h.svc.SetMuted(context.Background(), 2, 3, 1, true)

		require.ErrorIs(t, err, httpx.ErrRecordNotFound)
		h.state.AssertNotCalled(t, "SetMuted", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})
}

func TestService_SetHandRaised(t *testing.T) {
	t.Run("success: user raises their hand", func(t *testing.T) {
		h := newHarness()
		h.repo.On("FindAnyBy", uint(1), uint(2)).Return(&RoomUser{IsBlocked: false}, nil)
		h.state.On("SetHandRaised", mock.Anything, uint(2), uint(1), true).Return(nil)

		err := h.svc.SetHandRaised(context.Background(), 2, 1, true)

		require.NoError(t, err)
		h.assertAllExpectations(t)
	})

	t.Run("failure: blocked user cannot raise hand", func(t *testing.T) {
		h := newHarness()
		h.repo.On("FindAnyBy", uint(1), uint(2)).Return(&RoomUser{IsBlocked: true}, nil)

		err := h.svc.SetHandRaised(context.Background(), 2, 1, true)

		require.ErrorIs(t, err, httpx.ErrUserBlocked)
		h.state.AssertNotCalled(t, "SetHandRaised", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: non-member is forbidden", func(t *testing.T) {
		h := newHarness()
		h.repo.On("FindAnyBy", uint(1), uint(2)).Return(nil, nil)

		err := h.svc.SetHandRaised(context.Background(), 2, 1, true)

		require.ErrorIs(t, err, httpx.ErrForbidden)
		h.state.AssertNotCalled(t, "SetHandRaised", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: membership lookup error is propagated", func(t *testing.T) {
		h := newHarness()
		lookupErr := errors.New("db down")
		h.repo.On("FindAnyBy", uint(1), uint(2)).Return(nil, lookupErr)

		err := h.svc.SetHandRaised(context.Background(), 2, 1, true)

		require.ErrorIs(t, err, lookupErr)
		h.state.AssertNotCalled(t, "SetHandRaised", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})
}

func TestService_FindByWithState(t *testing.T) {
	t.Run("success: returns room user with mute and hand-raised state", func(t *testing.T) {
		h := newHarness()
		ru := &RoomUser{UserID: 1, RoomID: 2}
		h.repo.On("FindBy", uint(1), uint(2)).Return(ru, nil)
		h.state.On("GetParticipantStates", mock.Anything, uint(2), []uint{1}).
			Return(map[uint]*roomstate.ParticipantState{1: {UserID: 1, IsMuted: true, IsHandRaised: true}}, nil)

		got, err := h.svc.FindByWithState(context.Background(), 1, 2)

		require.NoError(t, err)
		assert.Same(t, ru, got)
		assert.True(t, got.IsMuted)
		assert.True(t, got.IsHandRaised)
		h.assertAllExpectations(t)
	})

	t.Run("success: member without state keeps zero values", func(t *testing.T) {
		h := newHarness()
		ru := &RoomUser{UserID: 1, RoomID: 2}
		h.repo.On("FindBy", uint(1), uint(2)).Return(ru, nil)
		h.state.On("GetParticipantStates", mock.Anything, uint(2), []uint{1}).
			Return(map[uint]*roomstate.ParticipantState{1: nil}, nil)

		got, err := h.svc.FindByWithState(context.Background(), 1, 2)

		require.NoError(t, err)
		assert.Same(t, ru, got)
		assert.False(t, got.IsMuted)
		assert.False(t, got.IsHandRaised)
		h.assertAllExpectations(t)
	})

	t.Run("success: nil result (not a member) skips state lookup", func(t *testing.T) {
		h := newHarness()
		h.repo.On("FindBy", uint(1), uint(2)).Return(nil, nil)

		got, err := h.svc.FindByWithState(context.Background(), 1, 2)

		require.NoError(t, err)
		assert.Nil(t, got)
		h.state.AssertNotCalled(t, "GetParticipantStates", mock.Anything, mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: repo error is propagated", func(t *testing.T) {
		h := newHarness()
		findErr := errors.New("db down")
		h.repo.On("FindBy", uint(1), uint(2)).Return(nil, findErr)

		got, err := h.svc.FindByWithState(context.Background(), 1, 2)

		require.Nil(t, got)
		require.ErrorIs(t, err, findErr)
		h.state.AssertNotCalled(t, "GetParticipantStates", mock.Anything, mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: state fetch error is propagated", func(t *testing.T) {
		h := newHarness()
		stateErr := errors.New("redis down")
		h.repo.On("FindBy", uint(1), uint(2)).Return(&RoomUser{UserID: 1, RoomID: 2}, nil)
		h.state.On("GetParticipantStates", mock.Anything, uint(2), []uint{1}).Return(nil, stateErr)

		got, err := h.svc.FindByWithState(context.Background(), 1, 2)

		require.Nil(t, got)
		require.ErrorIs(t, err, stateErr)
		h.assertAllExpectations(t)
	})
}

func TestService_ListRaisedHands(t *testing.T) {
	t.Run("success: returns raised-hand users with state and total count", func(t *testing.T) {
		h := newHarness()
		p := listopts.Pagination{Page: 1, PageSize: 10}
		users := []RoomUser{{UserID: 7}, {UserID: 9}}
		h.repo.On("FindAnyBy", uint(1), uint(4)).Return(&RoomUser{IsBlocked: false}, nil)
		h.state.On("GetRaisedHands", mock.Anything, uint(4), p).Return([]uint{7, 9}, nil)
		h.state.On("CountRaisedHands", mock.Anything, uint(4)).Return(int64(2), nil)
		h.repo.On("ListByUserIDs", uint(4), []uint{7, 9}).Return(users, nil)
		h.state.On("GetParticipantStates", mock.Anything, uint(4), []uint{7, 9}).
			Return(map[uint]*roomstate.ParticipantState{
				7: {UserID: 7, IsMuted: true, IsHandRaised: true},
				9: {UserID: 9, IsHandRaised: true},
			}, nil)

		got, count, err := h.svc.ListRaisedHands(context.Background(), 4, 1, p)

		require.NoError(t, err)
		assert.Equal(t, int64(2), count)
		require.Len(t, got, 2)
		assert.True(t, got[0].IsHandRaised)
		assert.True(t, got[0].IsMuted)
		assert.False(t, got[1].IsMuted)
		h.assertAllExpectations(t)
	})

	t.Run("success: pagination params are passed through to the state repo", func(t *testing.T) {
		h := newHarness()
		p := listopts.Pagination{Page: 2, PageSize: 2}
		h.repo.On("FindAnyBy", uint(1), uint(4)).Return(&RoomUser{IsBlocked: false}, nil)
		h.state.On("GetRaisedHands", mock.Anything, uint(4), p).Return([]uint{7, 9}, nil)
		h.state.On("CountRaisedHands", mock.Anything, uint(4)).Return(int64(5), nil)
		h.repo.On("ListByUserIDs", uint(4), []uint{7, 9}).Return([]RoomUser{{UserID: 7}, {UserID: 9}}, nil)
		h.state.On("GetParticipantStates", mock.Anything, uint(4), []uint{7, 9}).
			Return(map[uint]*roomstate.ParticipantState{}, nil)

		got, count, err := h.svc.ListRaisedHands(context.Background(), 4, 1, p)

		require.NoError(t, err)
		assert.Equal(t, int64(5), count)
		require.Len(t, got, 2)
		h.assertAllExpectations(t)
	})

	t.Run("success: no raised hands returns empty list without repo lookup", func(t *testing.T) {
		h := newHarness()
		p := listopts.Pagination{Page: 1, PageSize: 10}
		h.repo.On("FindAnyBy", uint(1), uint(4)).Return(&RoomUser{IsBlocked: false}, nil)
		h.state.On("GetRaisedHands", mock.Anything, uint(4), p).Return([]uint{}, nil)
		h.state.On("CountRaisedHands", mock.Anything, uint(4)).Return(int64(0), nil)
		h.repo.On("ListByUserIDs", uint(4), []uint{}).Return([]RoomUser{}, nil)

		got, count, err := h.svc.ListRaisedHands(context.Background(), 4, 1, p)

		require.NoError(t, err)
		assert.Zero(t, count)
		assert.Empty(t, got)
		h.state.AssertNotCalled(t, "GetParticipantStates", mock.Anything, mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: non-member gets ErrForbidden before any lookup", func(t *testing.T) {
		h := newHarness()
		p := listopts.Pagination{Page: 1, PageSize: 10}
		h.repo.On("FindAnyBy", uint(1), uint(4)).Return(nil, nil)

		got, count, err := h.svc.ListRaisedHands(context.Background(), 4, 1, p)

		require.Nil(t, got)
		require.Zero(t, count)
		require.ErrorIs(t, err, httpx.ErrForbidden)
		h.state.AssertNotCalled(t, "GetRaisedHands", mock.Anything, mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: blocked user gets ErrUserBlocked before any lookup", func(t *testing.T) {
		h := newHarness()
		p := listopts.Pagination{Page: 1, PageSize: 10}
		h.repo.On("FindAnyBy", uint(1), uint(4)).Return(&RoomUser{IsBlocked: true}, nil)

		got, count, err := h.svc.ListRaisedHands(context.Background(), 4, 1, p)

		require.Nil(t, got)
		require.Zero(t, count)
		require.ErrorIs(t, err, httpx.ErrUserBlocked)
		h.state.AssertNotCalled(t, "GetRaisedHands", mock.Anything, mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: GetRaisedHands error is propagated", func(t *testing.T) {
		h := newHarness()
		p := listopts.Pagination{Page: 1, PageSize: 10}
		stateErr := errors.New("redis down")
		h.repo.On("FindAnyBy", uint(1), uint(4)).Return(&RoomUser{IsBlocked: false}, nil)
		h.state.On("GetRaisedHands", mock.Anything, uint(4), p).Return(nil, stateErr)

		got, count, err := h.svc.ListRaisedHands(context.Background(), 4, 1, p)

		require.Nil(t, got)
		require.Zero(t, count)
		require.ErrorIs(t, err, stateErr)
		h.repo.AssertNotCalled(t, "ListByUserIDs", mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: ListByUserIDs error is propagated", func(t *testing.T) {
		h := newHarness()
		p := listopts.Pagination{Page: 1, PageSize: 10}
		repoErr := errors.New("db down")
		h.repo.On("FindAnyBy", uint(1), uint(4)).Return(&RoomUser{IsBlocked: false}, nil)
		h.state.On("GetRaisedHands", mock.Anything, uint(4), p).Return([]uint{7}, nil)
		h.state.On("CountRaisedHands", mock.Anything, uint(4)).Return(int64(1), nil)
		h.repo.On("ListByUserIDs", uint(4), []uint{7}).Return(nil, repoErr)

		got, count, err := h.svc.ListRaisedHands(context.Background(), 4, 1, p)

		require.Nil(t, got)
		require.Zero(t, count)
		require.ErrorIs(t, err, repoErr)
		h.assertAllExpectations(t)
	})
}

func TestService_Block(t *testing.T) {
	t.Run("success: blocks user, demotes to listener role and revokes publishing", func(t *testing.T) {
		h := newHarness()
		target := &RoomUser{UserID: 3, Role: role.Role{Name: role.RoleSpeaker}}
		h.repo.On("FindBy", uint(1), uint(2)).Return(&RoomUser{Role: role.Role{Name: role.RoleAdmin}}, nil)
		h.repo.On("FindAnyBy", uint(3), uint(2)).Return(target, nil)
		h.roles.On("FindByName", role.RoleListener).Return(&role.Role{BaseModel: model.BaseModel{ID: 4}, Name: role.RoleListener}, nil)
		h.repo.On("Block", uint(2), uint(3), uint(1), uint(4)).Return(nil)
		h.revoker.On("RevokePublishing", uint(2), uint(3)).Return()

		got, err := h.svc.Block(2, 3, 1)

		require.NoError(t, err)
		assert.Same(t, target, got)
		h.assertAllExpectations(t)
	})

	t.Run("failure: actor cannot block themselves", func(t *testing.T) {
		h := newHarness()

		got, err := h.svc.Block(2, 1, 1)

		require.Nil(t, got)
		require.ErrorIs(t, err, httpx.ErrForbidden)
		h.repo.AssertNotCalled(t, "FindBy", mock.Anything, mock.Anything)
		h.repo.AssertNotCalled(t, "Block", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: actor without moderation rights gets ErrForbidden", func(t *testing.T) {
		h := newHarness()
		h.repo.On("FindBy", uint(1), uint(2)).Return(&RoomUser{Role: role.Role{Name: role.RoleModerator}}, nil)
		h.repo.On("FindAnyBy", uint(3), uint(2)).Return(&RoomUser{Role: role.Role{Name: role.RoleAdmin}}, nil)

		got, err := h.svc.Block(2, 3, 1)

		require.Nil(t, got)
		require.ErrorIs(t, err, httpx.ErrForbidden)
		h.roles.AssertNotCalled(t, "FindByName", mock.Anything)
		h.repo.AssertNotCalled(t, "Block", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: target not in room returns ErrRecordNotFound", func(t *testing.T) {
		h := newHarness()
		h.repo.On("FindBy", uint(1), uint(2)).Return(&RoomUser{Role: role.Role{Name: role.RoleAdmin}}, nil)
		h.repo.On("FindAnyBy", uint(3), uint(2)).Return(nil, nil)

		got, err := h.svc.Block(2, 3, 1)

		require.Nil(t, got)
		require.ErrorIs(t, err, httpx.ErrRecordNotFound)
		h.repo.AssertNotCalled(t, "Block", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: listener role lookup error is propagated, Block never called", func(t *testing.T) {
		h := newHarness()
		h.repo.On("FindBy", uint(1), uint(2)).Return(&RoomUser{Role: role.Role{Name: role.RoleAdmin}}, nil)
		h.repo.On("FindAnyBy", uint(3), uint(2)).Return(&RoomUser{Role: role.Role{Name: role.RoleSpeaker}}, nil)
		roleErr := errors.New("role not found")
		h.roles.On("FindByName", role.RoleListener).Return(nil, roleErr)

		got, err := h.svc.Block(2, 3, 1)

		require.Nil(t, got)
		require.ErrorIs(t, err, roleErr)
		h.repo.AssertNotCalled(t, "Block", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		h.revoker.AssertNotCalled(t, "RevokePublishing", mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: Block error is propagated, publishing not revoked", func(t *testing.T) {
		h := newHarness()
		h.repo.On("FindBy", uint(1), uint(2)).Return(&RoomUser{Role: role.Role{Name: role.RoleAdmin}}, nil)
		h.repo.On("FindAnyBy", uint(3), uint(2)).Return(&RoomUser{Role: role.Role{Name: role.RoleSpeaker}}, nil)
		h.roles.On("FindByName", role.RoleListener).Return(&role.Role{BaseModel: model.BaseModel{ID: 4}, Name: role.RoleListener}, nil)
		blockErr := errors.New("update failed")
		h.repo.On("Block", uint(2), uint(3), uint(1), uint(4)).Return(blockErr)

		got, err := h.svc.Block(2, 3, 1)

		require.Nil(t, got)
		require.ErrorIs(t, err, blockErr)
		h.revoker.AssertNotCalled(t, "RevokePublishing", mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})
}

func TestService_Unblock(t *testing.T) {
	t.Run("success: admin unblocks a blocked user", func(t *testing.T) {
		h := newHarness()
		h.repo.On("FindBy", uint(1), uint(2)).Return(&RoomUser{Role: role.Role{Name: role.RoleAdmin}}, nil)
		h.repo.On("FindAnyBy", uint(3), uint(2)).Return(&RoomUser{Role: role.Role{Name: role.RoleListener}, IsBlocked: true}, nil)
		h.repo.On("Unblock", uint(2), uint(3)).Return(nil)

		err := h.svc.Unblock(2, 3, 1)

		require.NoError(t, err)
		h.assertAllExpectations(t)
	})

	t.Run("failure: target not blocked returns ErrRecordNotFound", func(t *testing.T) {
		h := newHarness()
		h.repo.On("FindBy", uint(1), uint(2)).Return(&RoomUser{Role: role.Role{Name: role.RoleAdmin}}, nil)
		h.repo.On("FindAnyBy", uint(3), uint(2)).Return(&RoomUser{Role: role.Role{Name: role.RoleListener}, IsBlocked: false}, nil)

		err := h.svc.Unblock(2, 3, 1)

		require.ErrorIs(t, err, httpx.ErrRecordNotFound)
		h.repo.AssertNotCalled(t, "Unblock", mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: actor not in room is forbidden", func(t *testing.T) {
		h := newHarness()
		h.repo.On("FindBy", uint(1), uint(2)).Return(nil, nil)

		err := h.svc.Unblock(2, 3, 1)

		require.ErrorIs(t, err, httpx.ErrForbidden)
		h.repo.AssertNotCalled(t, "Unblock", mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: Unblock error is propagated", func(t *testing.T) {
		h := newHarness()
		h.repo.On("FindBy", uint(1), uint(2)).Return(&RoomUser{Role: role.Role{Name: role.RoleAdmin}}, nil)
		h.repo.On("FindAnyBy", uint(3), uint(2)).Return(&RoomUser{Role: role.Role{Name: role.RoleListener}, IsBlocked: true}, nil)
		unblockErr := errors.New("update failed")
		h.repo.On("Unblock", uint(2), uint(3)).Return(unblockErr)

		err := h.svc.Unblock(2, 3, 1)

		require.ErrorIs(t, err, unblockErr)
		h.assertAllExpectations(t)
	})
}

func TestService_ListBlockedByRoomID(t *testing.T) {
	t.Run("success: returns blocked users and filtered count", func(t *testing.T) {
		h := newHarness()
		filter := RoomUserFilter{Query: "alice"}
		p := listopts.Pagination{Page: 1, PageSize: 10}
		users := []RoomUser{{UserID: 7, IsBlocked: true}}

		h.repo.On("FindBy", uint(1), uint(4)).Return(&RoomUser{Role: role.Role{Name: role.RoleModerator}}, nil)
		h.repo.On("ListBlockedByRoomID", uint(4), filter, p).Return(users, nil)
		h.repo.On("CountBlockedByRoomID", uint(4), filter).Return(int64(1), nil)

		got, count, err := h.svc.ListBlockedByRoomID(4, 1, filter, p)

		require.NoError(t, err)
		assert.Equal(t, users, got)
		assert.Equal(t, int64(1), count)
		h.assertAllExpectations(t)
	})

	t.Run("failure: actor without manage rights gets ErrForbidden", func(t *testing.T) {
		h := newHarness()
		filter := RoomUserFilter{}
		p := listopts.Pagination{Page: 1, PageSize: 10}
		h.repo.On("FindBy", uint(1), uint(4)).Return(&RoomUser{Role: role.Role{Name: role.RoleListener}}, nil)

		got, count, err := h.svc.ListBlockedByRoomID(4, 1, filter, p)

		require.Nil(t, got)
		require.Zero(t, count)
		require.ErrorIs(t, err, httpx.ErrForbidden)
		h.repo.AssertNotCalled(t, "ListBlockedByRoomID", mock.Anything, mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: non-member gets ErrForbidden before repo is called", func(t *testing.T) {
		h := newHarness()
		filter := RoomUserFilter{}
		p := listopts.Pagination{Page: 1, PageSize: 10}
		h.repo.On("FindBy", uint(1), uint(4)).Return(nil, nil)

		got, count, err := h.svc.ListBlockedByRoomID(4, 1, filter, p)

		require.Nil(t, got)
		require.Zero(t, count)
		require.ErrorIs(t, err, httpx.ErrForbidden)
		h.repo.AssertNotCalled(t, "ListBlockedByRoomID", mock.Anything, mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: List error short-circuits before Count", func(t *testing.T) {
		h := newHarness()
		filter := RoomUserFilter{}
		p := listopts.Pagination{Page: 1, PageSize: 10}
		listErr := errors.New("query failed")
		h.repo.On("FindBy", uint(1), uint(4)).Return(&RoomUser{Role: role.Role{Name: role.RoleModerator}}, nil)
		h.repo.On("ListBlockedByRoomID", uint(4), filter, p).Return(nil, listErr)

		got, count, err := h.svc.ListBlockedByRoomID(4, 1, filter, p)

		require.Nil(t, got)
		require.Zero(t, count)
		require.ErrorIs(t, err, listErr)
		h.repo.AssertNotCalled(t, "CountBlockedByRoomID", mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: Count error after successful List still fails the call", func(t *testing.T) {
		h := newHarness()
		filter := RoomUserFilter{}
		p := listopts.Pagination{Page: 1, PageSize: 10}
		users := []RoomUser{{UserID: 7, IsBlocked: true}}
		countErr := errors.New("count failed")
		h.repo.On("FindBy", uint(1), uint(4)).Return(&RoomUser{Role: role.Role{Name: role.RoleModerator}}, nil)
		h.repo.On("ListBlockedByRoomID", uint(4), filter, p).Return(users, nil)
		h.repo.On("CountBlockedByRoomID", uint(4), filter).Return(int64(0), countErr)

		got, count, err := h.svc.ListBlockedByRoomID(4, 1, filter, p)

		require.Nil(t, got)
		require.Zero(t, count)
		require.ErrorIs(t, err, countErr)
		h.assertAllExpectations(t)
	})
}
