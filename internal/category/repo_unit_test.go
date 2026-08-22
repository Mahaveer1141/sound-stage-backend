package category

import (
	"database/sql/driver"
	"testing"

	"sound-stage-backend/internal/pkg/testutil"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepo_List_Unit(t *testing.T) {
	t.Run("returns list", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`SELECT \* FROM "categories`,
		).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "description"}).
				AddRow(1, driver.Value("Music"), nil))

		got, err := repo.List()

		require.NoError(t, err)
		assert.Equal(t, 1, len(got))
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
