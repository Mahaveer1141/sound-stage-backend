package role

import (
	"sound-stage-backend/internal/pkg/testutil"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func seedRoles(t *testing.T, db *gorm.DB, names ...RoleName) {
	t.Helper()
	for _, n := range names {
		require.NoError(t, db.Create(&Role{Name: string(n)}).Error)
	}
}

func TestRepo_FindByName_Integration(t *testing.T) {
	t.Run("finds an existing seeded role", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &Role{})
		seedRoles(t, db, RoleOwner, RoleAdmin, RoleModerator, RoleSpeaker, RoleListener)
		repo := NewRepo(db)

		got, err := repo.FindByName(RoleAdmin)

		require.NoError(t, err)
		require.NotNil(t, got)
		require.Equal(t, string(RoleAdmin), got.Name)
		require.NotZero(t, got.ID)
	})

	t.Run("returns gorm.ErrRecordNotFound for a role that does not exist", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &Role{})
		seedRoles(t, db, RoleAdmin)
		repo := NewRepo(db)

		got, err := repo.FindByName(RoleModerator)

		require.Nil(t, got)
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("empty database returns not found for any name", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &Role{})
		repo := NewRepo(db)

		got, err := repo.FindByName(RoleAdmin)

		require.Nil(t, got)
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("is case-sensitive on name lookup", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &Role{})
		seedRoles(t, db, RoleAdmin)
		repo := NewRepo(db)

		got, err := repo.FindByName(RoleName("Admin"))

		require.Nil(t, got)
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("returns correct role among multiple distinct roles", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &Role{})
		seedRoles(t, db, RoleAdmin, RoleModerator)
		repo := NewRepo(db)

		got, err := repo.FindByName(RoleModerator)

		require.NoError(t, err)
		require.Equal(t, string(RoleModerator), got.Name)
	})

	t.Run("unique index prevents duplicate role names", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &Role{})
		seedRoles(t, db, RoleAdmin)

		err := db.Create(&Role{Name: string(RoleAdmin)}).Error

		require.Error(t, err)
	})

	t.Run("closed/broken connection surfaces as an error, not a panic", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &Role{})
		repo := NewRepo(db)

		sqlDB, err := db.DB()
		require.NoError(t, err)
		require.NoError(t, sqlDB.Close())

		require.NotPanics(t, func() {
			got, err := repo.FindByName(RoleAdmin)
			require.Nil(t, got)
			require.Error(t, err)
		})
	})
}
