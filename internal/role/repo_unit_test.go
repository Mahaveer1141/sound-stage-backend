package role

import (
	"database/sql/driver"
	"errors"
	"regexp"
	"sound-stage-backend/internal/pkg/testutil"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestRepo_FindByName_Unit(t *testing.T) {
	roleCols := []string{"id", "name"}

	t.Run("returns role when found", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "roles" WHERE name = $1 ORDER BY "roles"."id" LIMIT $2`)).
			WithArgs(driver.Value(RoleAdmin), driver.Value(1)).
			WillReturnRows(sqlmock.NewRows(roleCols).AddRow(1, "admin"))

		got, err := repo.FindByName(RoleAdmin)

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, uint(1), got.ID)
		assert.Equal(t, string(RoleAdmin), got.Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns ErrRecordNotFound when no row matches", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "roles" WHERE name = $1 ORDER BY "roles"."id" LIMIT $2`)).
			WithArgs(driver.Value(RoleName("nonexistent")), driver.Value(1)).
			WillReturnRows(sqlmock.NewRows(roleCols))

		got, err := repo.FindByName(RoleName("nonexistent"))

		require.Nil(t, got)
		require.Error(t, err)
		assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates underlying db error", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		dbErr := errors.New("connection reset by peer")
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "roles" WHERE name = $1 ORDER BY "roles"."id" LIMIT $2`)).
			WithArgs(driver.Value(RoleAdmin), driver.Value(1)).
			WillReturnError(dbErr)

		got, err := repo.FindByName(RoleAdmin)

		require.Nil(t, got)
		require.Error(t, err)
		assert.ErrorContains(t, err, "connection reset by peer")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty name still issues a query and can miss", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "roles" WHERE name = $1 ORDER BY "roles"."id" LIMIT $2`)).
			WithArgs(driver.Value(RoleName("")), driver.Value(1)).
			WillReturnRows(sqlmock.NewRows(roleCols))

		got, err := repo.FindByName(RoleName(""))

		require.Nil(t, got)
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
