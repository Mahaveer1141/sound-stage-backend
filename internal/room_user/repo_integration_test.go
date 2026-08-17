package roomuser

import (
	"testing"

	"sound-stage-backend/internal/pkg/listopts"
	"sound-stage-backend/internal/pkg/testutil"
	"sound-stage-backend/internal/role"
	"sound-stage-backend/internal/user"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func seedRoles(t *testing.T, db *gorm.DB) {
	t.Helper()
	for _, rn := range []role.RoleName{role.RoleOwner, role.RoleAdmin, role.RoleListener, role.RoleSpeaker, role.RoleModerator} {
		require.NoError(t, db.Create(&role.Role{Name: string(rn)}).Error)
	}
}

type testDeps struct {
	roomID uint
	owner  user.User
	admin  user.User
	repo   *Repo
}

func setupRoomUserTest(t *testing.T, db *gorm.DB) testDeps {
	t.Helper()
	seedRoles(t, db)

	owner := user.User{Email: "owner@example.com", FirstName: "Owner"}
	require.NoError(t, db.Create(&owner).Error)

	admin := user.User{Email: "admin@example.com", FirstName: "Admin"}
	require.NoError(t, db.Create(&admin).Error)

	return testDeps{roomID: 1, owner: owner, admin: admin, repo: NewRepo(db)}
}

func TestRepo_Create_Integration(t *testing.T) {
	t.Run("persists a room user", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &RoomUser{}, &user.User{}, &role.Role{})
		deps := setupRoomUserTest(t, db)

		var listener role.Role
		require.NoError(t, db.Where("name = ?", string(role.RoleListener)).First(&listener).Error)

		got, err := deps.repo.Create(db, deps.owner.ID, deps.roomID, listener.ID)
		require.NoError(t, err)
		require.NotNil(t, got)
		require.Equal(t, deps.owner.ID, got.UserID)
		require.Equal(t, deps.roomID, got.RoomID)
		require.Equal(t, listener.ID, got.RoleID)

		var fetched RoomUser
		require.NoError(t, db.First(&fetched, got.ID).Error)
		require.Equal(t, got.ID, fetched.ID)
	})
}

func TestRepo_FindBy_Integration(t *testing.T) {
	t.Run("finds room user with preloaded user and role", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &RoomUser{}, &user.User{}, &role.Role{})
		deps := setupRoomUserTest(t, db)

		var listener role.Role
		require.NoError(t, db.Where("name = ?", string(role.RoleListener)).First(&listener).Error)

		created, err := deps.repo.Create(db, deps.admin.ID, deps.roomID, listener.ID)
		require.NoError(t, err)

		got, err := deps.repo.FindBy(deps.admin.ID, deps.roomID)
		require.NoError(t, err)
		require.NotNil(t, got)
		require.Equal(t, created.ID, got.ID)
		require.Equal(t, deps.admin.Email, got.User.Email)
		require.Equal(t, string(role.RoleListener), got.Role.Name)
	})

	t.Run("returns nil when not found", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &RoomUser{}, &user.User{}, &role.Role{})
		deps := setupRoomUserTest(t, db)

		got, err := deps.repo.FindBy(999, deps.roomID)
		require.NoError(t, err)
		require.Nil(t, got)
	})
}

func TestRepo_UpdateActivity_Integration(t *testing.T) {
	t.Run("updates join and leave timestamps and online status", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &RoomUser{}, &user.User{}, &role.Role{})
		deps := setupRoomUserTest(t, db)

		var listener role.Role
		require.NoError(t, db.Where("name = ?", string(role.RoleListener)).First(&listener).Error)

		ru, err := deps.repo.Create(db, deps.admin.ID, deps.roomID, listener.ID)
		require.NoError(t, err)

		require.NoError(t, deps.repo.UpdateActivity(ru, ActivityJoin))
		require.NoError(t, db.First(ru, ru.ID).Error)
		require.True(t, ru.IsOnline)
		require.False(t, ru.LastJoinedAt.IsZero())

		require.NoError(t, deps.repo.UpdateActivity(ru, ActivityLeave))
		require.NoError(t, db.First(ru, ru.ID).Error)
		require.False(t, ru.IsOnline)
		require.False(t, ru.LastLeftAt.IsZero())
	})
}

func TestRepo_HasRoles_Integration(t *testing.T) {
	t.Run("returns true when user has one of the requested roles", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &RoomUser{}, &user.User{}, &role.Role{})
		deps := setupRoomUserTest(t, db)

		var listener role.Role
		require.NoError(t, db.Where("name = ?", string(role.RoleListener)).First(&listener).Error)

		_, err := deps.repo.Create(db, deps.admin.ID, deps.roomID, listener.ID)
		require.NoError(t, err)

		got, err := deps.repo.HasRoles(deps.admin.ID, deps.roomID, []role.RoleName{role.RoleOwner, role.RoleListener})
		require.NoError(t, err)
		require.True(t, got)
	})

	t.Run("returns false when user has none of the requested roles", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &RoomUser{}, &user.User{}, &role.Role{})
		deps := setupRoomUserTest(t, db)

		var listener role.Role
		require.NoError(t, db.Where("name = ?", string(role.RoleListener)).First(&listener).Error)

		_, err := deps.repo.Create(db, deps.admin.ID, deps.roomID, listener.ID)
		require.NoError(t, err)

		got, err := deps.repo.HasRoles(deps.admin.ID, deps.roomID, []role.RoleName{role.RoleOwner})
		require.NoError(t, err)
		require.False(t, got)
	})
}

func TestRepo_ListByRoomID_Integration(t *testing.T) {
	t.Run("lists users with sort and pagination", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &RoomUser{}, &user.User{}, &role.Role{})
		deps := setupRoomUserTest(t, db)

		var listener, speaker role.Role
		require.NoError(t, db.Where("name = ?", string(role.RoleListener)).First(&listener).Error)
		require.NoError(t, db.Where("name = ?", string(role.RoleSpeaker)).First(&speaker).Error)

		u1 := user.User{Email: "u1@example.com", FirstName: "U1"}
		u2 := user.User{Email: "u2@example.com", FirstName: "U2"}
		u3 := user.User{Email: "u3@example.com", FirstName: "U3"}
		require.NoError(t, db.Create(&u1).Error)
		require.NoError(t, db.Create(&u2).Error)
		require.NoError(t, db.Create(&u3).Error)

		_, err := deps.repo.Create(db, u1.ID, deps.roomID, listener.ID)
		require.NoError(t, err)
		_, err = deps.repo.Create(db, u2.ID, deps.roomID, speaker.ID)
		require.NoError(t, err)
		_, err = deps.repo.Create(db, u3.ID, deps.roomID, listener.ID)
		require.NoError(t, err)

		got, err := deps.repo.ListByRoomID(
			deps.roomID,
			RoomUserFilter{},
			listopts.Sort{Field: "created_at", Order: "asc"},
			listopts.Pagination{Page: 2, PageSize: 1},
		)

		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, u2.ID, got[0].User.ID)
		require.Equal(t, string(role.RoleSpeaker), got[0].Role.Name)
	})

	t.Run("filters room users by role", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &RoomUser{}, &user.User{}, &role.Role{})
		deps := setupRoomUserTest(t, db)

		var listener, speaker role.Role
		require.NoError(t, db.Where("name = ?", string(role.RoleListener)).First(&listener).Error)
		require.NoError(t, db.Where("name = ?", string(role.RoleSpeaker)).First(&speaker).Error)

		u1 := user.User{Email: "u1@example.com", FirstName: "U1"}
		u2 := user.User{Email: "u2@example.com", FirstName: "U2"}
		require.NoError(t, db.Create(&u1).Error)
		require.NoError(t, db.Create(&u2).Error)

		_, err := deps.repo.Create(db, u1.ID, deps.roomID, listener.ID)
		require.NoError(t, err)
		_, err = deps.repo.Create(db, u2.ID, deps.roomID, speaker.ID)
		require.NoError(t, err)

		got, err := deps.repo.ListByRoomID(
			deps.roomID,
			RoomUserFilter{Roles: []string{string(role.RoleListener)}},
			listopts.Sort{Field: "created_at", Order: "asc"},
			listopts.Pagination{Page: 1, PageSize: 10},
		)

		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, u1.ID, got[0].User.ID)
		require.Equal(t, string(role.RoleListener), got[0].Role.Name)
	})
}

func TestRepo_CountByRoomID_Integration(t *testing.T) {
	t.Run("returns total and filtered counts", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &RoomUser{}, &user.User{}, &role.Role{})
		deps := setupRoomUserTest(t, db)

		var listener, speaker role.Role
		require.NoError(t, db.Where("name = ?", string(role.RoleListener)).First(&listener).Error)
		require.NoError(t, db.Where("name = ?", string(role.RoleSpeaker)).First(&speaker).Error)

		u1 := user.User{Email: "u1@example.com", FirstName: "U1"}
		u2 := user.User{Email: "u2@example.com", FirstName: "U2"}
		require.NoError(t, db.Create(&u1).Error)
		require.NoError(t, db.Create(&u2).Error)

		_, err := deps.repo.Create(db, u1.ID, deps.roomID, listener.ID)
		require.NoError(t, err)
		_, err = deps.repo.Create(db, u2.ID, deps.roomID, speaker.ID)
		require.NoError(t, err)

		total, err := deps.repo.CountByRoomID(deps.roomID, RoomUserFilter{})
		require.NoError(t, err)
		require.Equal(t, int64(2), total)

		filtered, err := deps.repo.CountByRoomID(deps.roomID, RoomUserFilter{Roles: []string{string(role.RoleListener)}})
		require.NoError(t, err)
		require.Equal(t, int64(1), filtered)
	})
}

func TestRepo_UpdateRole_Integration(t *testing.T) {
	t.Run("updates role by id", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &RoomUser{}, &user.User{}, &role.Role{})
		deps := setupRoomUserTest(t, db)

		var listener, admin role.Role
		require.NoError(t, db.Where("name = ?", string(role.RoleListener)).First(&listener).Error)
		require.NoError(t, db.Where("name = ?", string(role.RoleAdmin)).First(&admin).Error)

		ru, err := deps.repo.Create(db, deps.admin.ID, deps.roomID, listener.ID)
		require.NoError(t, err)

		require.NoError(t, deps.repo.UpdateRole(deps.roomID, deps.admin.ID, admin.ID))

		require.NoError(t, db.First(ru, ru.ID).Error)
		require.Equal(t, admin.ID, ru.RoleID)
	})
}
