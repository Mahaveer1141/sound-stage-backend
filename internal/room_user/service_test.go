package roomuser

import (
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
func (m *mockRepo) UpdateActivity(ru *RoomUser, activity Activity) error {
	args := m.Called(ru, activity)
	return args.Error(0)
}
func (m *mockRepo) HasRoles(userID, roomID uint, permissions []role.RoleName) (bool, error) {
	args := m.Called(userID, roomID, permissions)
	return args.Bool(0), args.Error(1)
}
func (m *mockRepo) ListByRoomID(roomID uint, filter RoomUserFilter, sort listopts.Sort, p listopts.Pagination) ([]RoomUser, error) {
	args := m.Called(roomID, filter, sort, p)
	rus, _ := args.Get(0).([]RoomUser)
	return rus, args.Error(1)
}
func (m *mockRepo) CountByRoomID(roomID uint, filter RoomUserFilter) (int64, error) {
	args := m.Called(roomID, filter)
	return args.Get(0).(int64), args.Error(1)
}
func (m *mockRepo) UpdateRole(roomID, userID, roleID uint) error {
	args := m.Called(roomID, userID, roleID)
	return args.Error(0)
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

type harness struct {
	repo    *mockRepo
	roles   *mockRoleFinder
	revoker *mockRevoker
	svc     *Service
}

func newHarness() *harness {
	repo := new(mockRepo)
	roles := new(mockRoleFinder)
	revoker := new(mockRevoker)
	return &harness{
		repo:    repo,
		roles:   roles,
		revoker: revoker,
		svc:     NewService(repo, roles, revoker),
	}
}

func (h *harness) assertAllExpectations(t *testing.T) {
	t.Helper()
	h.repo.AssertExpectations(t)
	h.roles.AssertExpectations(t)
	h.revoker.AssertExpectations(t)
}

func TestService_AddUserWithTx(t *testing.T) {
	t.Run("success: existing member re-joining updates activity, no role lookup or create", func(t *testing.T) {
		h := newHarness()
		existing := &RoomUser{UserID: 1, RoomID: 2}
		h.repo.On("FindBy", uint(1), uint(2)).Return(existing, nil)
		h.repo.On("UpdateActivity", existing, ActivityJoin).Return(nil)

		got, err := h.svc.AddUserWithTx(nil, 1, 2, role.RoleListener)

		require.NoError(t, err)
		assert.Same(t, existing, got)
		h.roles.AssertNotCalled(t, "FindByName", mock.Anything)
		h.repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("success: new member with explicit role is created with that role", func(t *testing.T) {
		h := newHarness()
		h.repo.On("FindBy", uint(1), uint(2)).Return(nil, nil)
		h.roles.On("FindByName", role.RoleListener).Return(&role.Role{BaseModel: model.BaseModel{ID: 4}}, nil)
		created := &RoomUser{UserID: 1, RoomID: 2}
		h.repo.On("Create", (*gorm.DB)(nil), uint(1), uint(2), uint(4)).Return(created, nil)

		got, err := h.svc.AddUserWithTx(nil, 1, 2, role.RoleListener)

		require.NoError(t, err)
		assert.Same(t, created, got)
		h.assertAllExpectations(t)
	})

	t.Run("success: new member with empty role name defaults to RoleListener", func(t *testing.T) {
		h := newHarness()
		h.repo.On("FindBy", uint(1), uint(2)).Return(nil, nil)
		h.roles.On("FindByName", role.RoleListener).Return(&role.Role{BaseModel: model.BaseModel{ID: 4}}, nil)
		created := &RoomUser{UserID: 1, RoomID: 2}
		h.repo.On("Create", (*gorm.DB)(nil), uint(1), uint(2), uint(4)).Return(created, nil)

		got, err := h.svc.AddUserWithTx(nil, 1, 2, role.RoleName(""))

		require.NoError(t, err)
		assert.Same(t, created, got)
		h.assertAllExpectations(t)
	})

	t.Run("failure: FindBy error short-circuits before any role lookup or create", func(t *testing.T) {
		h := newHarness()
		findErr := errors.New("db down")
		h.repo.On("FindBy", uint(1), uint(2)).Return(nil, findErr)

		got, err := h.svc.AddUserWithTx(nil, 1, 2, role.RoleListener)

		require.Nil(t, got)
		require.ErrorIs(t, err, findErr)
		h.roles.AssertNotCalled(t, "FindByName", mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: UpdateActivity error on re-join is propagated", func(t *testing.T) {
		h := newHarness()
		existing := &RoomUser{UserID: 1, RoomID: 2}
		h.repo.On("FindBy", uint(1), uint(2)).Return(existing, nil)
		updateErr := errors.New("update failed")
		h.repo.On("UpdateActivity", existing, ActivityJoin).Return(updateErr)

		got, err := h.svc.AddUserWithTx(nil, 1, 2, role.RoleListener)

		require.Nil(t, got)
		require.ErrorIs(t, err, updateErr)
		h.assertAllExpectations(t)
	})

	t.Run("failure: role lookup error for new member is propagated, Create never called", func(t *testing.T) {
		h := newHarness()
		h.repo.On("FindBy", uint(1), uint(2)).Return(nil, nil)
		roleErr := errors.New("unknown role")
		h.roles.On("FindByName", role.RoleName("no-role")).Return(nil, roleErr)

		got, err := h.svc.AddUserWithTx(nil, 1, 2, role.RoleName("no-role"))

		require.Nil(t, got)
		require.ErrorIs(t, err, roleErr)
		h.repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: Create error for new member is propagated", func(t *testing.T) {
		h := newHarness()
		h.repo.On("FindBy", uint(1), uint(2)).Return(nil, nil)
		h.roles.On("FindByName", role.RoleListener).Return(&role.Role{BaseModel: model.BaseModel{ID: 4}}, nil)
		createErr := errors.New("insert failed")
		h.repo.On("Create", (*gorm.DB)(nil), uint(1), uint(2), uint(4)).Return(nil, createErr)

		got, err := h.svc.AddUserWithTx(nil, 1, 2, role.RoleListener)

		require.Nil(t, got)
		require.ErrorIs(t, err, createErr)
		h.assertAllExpectations(t)
	})
}

func TestService_AddUser(t *testing.T) {
	t.Run("success: delegates to AddUserWithTx with nil tx", func(t *testing.T) {
		h := newHarness()
		h.repo.On("FindBy", uint(1), uint(2)).Return(nil, nil)
		h.roles.On("FindByName", role.RoleListener).Return(&role.Role{BaseModel: model.BaseModel{ID: 4}}, nil)
		created := &RoomUser{UserID: 1, RoomID: 2}
		h.repo.On("Create", (*gorm.DB)(nil), uint(1), uint(2), uint(4)).Return(created, nil)

		got, err := h.svc.AddUser(1, 2, role.RoleListener)

		require.NoError(t, err)
		assert.Same(t, created, got)
		h.assertAllExpectations(t)
	})

	t.Run("failure: underlying error is propagated", func(t *testing.T) {
		h := newHarness()
		findErr := errors.New("db down")
		h.repo.On("FindBy", uint(1), uint(2)).Return(nil, findErr)

		got, err := h.svc.AddUser(1, 2, role.RoleListener)

		require.Nil(t, got)
		require.ErrorIs(t, err, findErr)
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

func TestService_RemoveUser(t *testing.T) {
	t.Run("success: marks activity as leave", func(t *testing.T) {
		h := newHarness()
		existing := &RoomUser{UserID: 1, RoomID: 2}
		h.repo.On("FindBy", uint(1), uint(2)).Return(existing, nil)
		h.repo.On("UpdateActivity", existing, ActivityLeave).Return(nil)

		err := h.svc.RemoveUser(1, 2)

		require.NoError(t, err)
		h.assertAllExpectations(t)
	})

	t.Run("failure: member not found returns ErrRecordNotFound", func(t *testing.T) {
		h := newHarness()
		h.repo.On("FindBy", uint(1), uint(2)).Return(nil, nil)

		err := h.svc.RemoveUser(1, 2)

		require.ErrorIs(t, err, httpx.ErrRecordNotFound)
		h.repo.AssertNotCalled(t, "UpdateActivity", mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: FindBy error is propagated", func(t *testing.T) {
		h := newHarness()
		findErr := errors.New("db down")
		h.repo.On("FindBy", uint(1), uint(2)).Return(nil, findErr)

		err := h.svc.RemoveUser(1, 2)

		require.ErrorIs(t, err, findErr)
		h.assertAllExpectations(t)
	})

	t.Run("failure: UpdateActivity error is propagated", func(t *testing.T) {
		h := newHarness()
		existing := &RoomUser{UserID: 1, RoomID: 2}
		h.repo.On("FindBy", uint(1), uint(2)).Return(existing, nil)
		updateErr := errors.New("update failed")
		h.repo.On("UpdateActivity", existing, ActivityLeave).Return(updateErr)

		err := h.svc.RemoveUser(1, 2)

		require.ErrorIs(t, err, updateErr)
		h.assertAllExpectations(t)
	})
}

func TestService_ListByRoomID(t *testing.T) {
	t.Run("success: returns users and total count", func(t *testing.T) {
		h := newHarness()
		filter := RoomUserFilter{}
		sort := listopts.Sort{}
		p := listopts.Pagination{Page: 1, PageSize: 10}
		users := []RoomUser{{UserID: 1}, {UserID: 2}}

		h.repo.On("ListByRoomID", uint(4), filter, sort, p).Return(users, nil)
		h.repo.On("CountByRoomID", uint(4), filter).Return(int64(2), nil)

		got, count, err := h.svc.ListByRoomID(4, filter, sort, p)

		require.NoError(t, err)
		assert.Equal(t, users, got)
		assert.Equal(t, int64(2), count)
		h.assertAllExpectations(t)
	})

	t.Run("failure: List error short-circuits before Count", func(t *testing.T) {
		h := newHarness()
		filter := RoomUserFilter{}
		sort := listopts.Sort{}
		p := listopts.Pagination{Page: 1, PageSize: 10}
		listErr := errors.New("query failed")
		h.repo.On("ListByRoomID", uint(4), filter, sort, p).Return(nil, listErr)

		got, count, err := h.svc.ListByRoomID(4, filter, sort, p)

		require.Nil(t, got)
		require.Zero(t, count)
		require.ErrorIs(t, err, listErr)
		h.repo.AssertNotCalled(t, "CountByRoomID", mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: Count error after successful List still fails the call", func(t *testing.T) {
		h := newHarness()
		filter := RoomUserFilter{}
		sort := listopts.Sort{}
		p := listopts.Pagination{Page: 1, PageSize: 10}
		users := []RoomUser{{UserID: 1}}
		countErr := errors.New("count failed")
		h.repo.On("ListByRoomID", uint(4), filter, sort, p).Return(users, nil)
		h.repo.On("CountByRoomID", uint(4), filter).Return(int64(0), countErr)

		got, count, err := h.svc.ListByRoomID(4, filter, sort, p)

		require.Nil(t, got)
		require.Zero(t, count)
		require.ErrorIs(t, err, countErr)
		h.assertAllExpectations(t)
	})
}

func TestService_UpdateRole(t *testing.T) {
	t.Run("success: promoting to non-listener role does not revoke publishing", func(t *testing.T) {
		h := newHarness()
		h.repo.On("HasRoles", uint(1), uint(2), role.RoleAssignmentPermissions[role.RoleSpeaker]).Return(true, nil)
		h.roles.On("FindByName", role.RoleSpeaker).Return(&role.Role{BaseModel: model.BaseModel{ID: 9}}, nil)
		h.repo.On("UpdateRole", uint(2), uint(3), uint(9)).Return(nil)

		err := h.svc.UpdateRole(2, 3, role.RoleSpeaker, 1)

		require.NoError(t, err)
		h.revoker.AssertNotCalled(t, "RevokePublishing", mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("success: demoting to listener revokes publishing", func(t *testing.T) {
		h := newHarness()
		h.repo.On("HasRoles", uint(1), uint(2), role.RoleAssignmentPermissions[role.RoleListener]).Return(true, nil)
		h.roles.On("FindByName", role.RoleListener).Return(&role.Role{BaseModel: model.BaseModel{ID: 3}}, nil)
		h.repo.On("UpdateRole", uint(2), uint(3), uint(3)).Return(nil)
		h.revoker.On("RevokePublishing", uint(2), uint(3)).Return()

		err := h.svc.UpdateRole(2, 3, role.RoleListener, 1)

		require.NoError(t, err)
		h.assertAllExpectations(t)
	})

	t.Run("failure: actor without permission gets ErrForbidden, nothing else runs", func(t *testing.T) {
		h := newHarness()
		h.repo.On("HasRoles", uint(1), uint(2), role.RoleAssignmentPermissions[role.RoleSpeaker]).Return(false, nil)

		err := h.svc.UpdateRole(2, 3, role.RoleSpeaker, 1)

		require.ErrorIs(t, err, httpx.ErrForbidden)
		h.roles.AssertNotCalled(t, "FindByName", mock.Anything)
		h.repo.AssertNotCalled(t, "UpdateRole", mock.Anything, mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: HasRoles error is propagated", func(t *testing.T) {
		h := newHarness()
		permErr := errors.New("permission check failed")
		h.repo.On("HasRoles", uint(1), uint(2), role.RoleAssignmentPermissions[role.RoleSpeaker]).Return(false, permErr)

		err := h.svc.UpdateRole(2, 3, role.RoleSpeaker, 1)

		require.ErrorIs(t, err, permErr)
		h.assertAllExpectations(t)
	})

	t.Run("failure: role lookup error is propagated, UpdateRole never called", func(t *testing.T) {
		h := newHarness()
		h.repo.On("HasRoles", uint(1), uint(2), role.RoleAssignmentPermissions[role.RoleSpeaker]).Return(true, nil)
		roleErr := errors.New("unknown role")
		h.roles.On("FindByName", role.RoleSpeaker).Return(nil, roleErr)

		err := h.svc.UpdateRole(2, 3, role.RoleSpeaker, 1)

		require.ErrorIs(t, err, roleErr)
		h.repo.AssertNotCalled(t, "UpdateRole", mock.Anything, mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})

	t.Run("failure: UpdateRole error is propagated, revoker never called", func(t *testing.T) {
		h := newHarness()
		h.repo.On("HasRoles", uint(1), uint(2), role.RoleAssignmentPermissions[role.RoleListener]).Return(true, nil)
		h.roles.On("FindByName", role.RoleListener).Return(&role.Role{BaseModel: model.BaseModel{ID: 3}}, nil)
		updateErr := errors.New("update failed")
		h.repo.On("UpdateRole", uint(2), uint(3), uint(3)).Return(updateErr)

		err := h.svc.UpdateRole(2, 3, role.RoleListener, 1)

		require.ErrorIs(t, err, updateErr)
		h.revoker.AssertNotCalled(t, "RevokePublishing", mock.Anything, mock.Anything)
		h.assertAllExpectations(t)
	})
}
