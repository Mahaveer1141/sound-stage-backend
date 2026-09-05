package roomuserfavourite

import (
	"regexp"
	"testing"

	"sound-stage-backend/internal/pkg/listopts"
	"sound-stage-backend/internal/pkg/testutil"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepo_Add_Unit(t *testing.T) {
	t.Run("creates a room user favourite", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(
			`INSERT INTO "room_user_favourites" ("created_at","updated_at","user_id","room_id") VALUES ($1,$2,$3,$4) ON CONFLICT DO NOTHING RETURNING "id"`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 20, 10).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
		mock.ExpectCommit()

		err := repo.Add(20, 10)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when insert fails", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(
			`INSERT INTO "room_user_favourites" ("created_at","updated_at","user_id","room_id") VALUES ($1,$2,$3,$4) ON CONFLICT DO NOTHING RETURNING "id"`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 20, 10).
			WillReturnError(assert.AnError)
		mock.ExpectRollback()

		err := repo.Add(20, 10)

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepo_Remove_Unit(t *testing.T) {
	t.Run("deletes a room user favourite", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			`DELETE FROM "room_user_favourites" WHERE user_id = $1 AND room_id = $2`)).
			WithArgs(20, 10).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := repo.Remove(20, 10)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when delete fails", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			`DELETE FROM "room_user_favourites" WHERE user_id = $1 AND room_id = $2`)).
			WithArgs(20, 10).
			WillReturnError(assert.AnError)
		mock.ExpectRollback()

		err := repo.Remove(20, 10)

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepo_ListUserFavourites_Unit(t *testing.T) {
	t.Run("returns error when query fails", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`SELECT \* FROM "room_user_favourites" WHERE user_id = \$1 AND \(room_id NOT IN \(SELECT room_id FROM room_users WHERE user_id = \$2 AND is_blocked = \$3\)\) ORDER BY id DESC LIMIT \$4`,
		).
			WithArgs(20, 20, true, 10).
			WillReturnError(assert.AnError)

		_, err := repo.ListUserFavourites(20, listopts.Pagination{Page: 1, PageSize: 10})

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepo_CountByUserID_Unit(t *testing.T) {
	t.Run("returns the number of favourites for a user", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`SELECT count\(\*\) FROM "room_user_favourites" WHERE user_id = \$1 AND \(room_id NOT IN \(SELECT room_id FROM room_users WHERE user_id = \$2 AND is_blocked = \$3\)\)`,
		).
			WithArgs(20, 20, true).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

		got, err := repo.CountByUserID(20)

		require.NoError(t, err)
		assert.Equal(t, int64(3), got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when count query fails", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`SELECT count\(\*\) FROM "room_user_favourites" WHERE user_id = \$1 AND \(room_id NOT IN \(SELECT room_id FROM room_users WHERE user_id = \$2 AND is_blocked = \$3\)\)`,
		).
			WithArgs(20, 20, true).
			WillReturnError(assert.AnError)

		_, err := repo.CountByUserID(20)

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepo_FindRoomIDsByUserID_Unit(t *testing.T) {
	t.Run("returns favourited room ids among the given rooms", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`SELECT "room_id" FROM "room_user_favourites" WHERE user_id = \$1 AND room_id IN \(\$2,\$3\)`,
		).
			WithArgs(20, 10, 11).
			WillReturnRows(sqlmock.NewRows([]string{"room_id"}).AddRow(10))

		got, err := repo.FindRoomIDsByUserID(20, []uint{10, 11})

		require.NoError(t, err)
		assert.Equal(t, []uint{10}, got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when query fails", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`SELECT "room_id" FROM "room_user_favourites" WHERE user_id = \$1 AND room_id IN \(\$2,\$3\)`,
		).
			WithArgs(20, 10, 11).
			WillReturnError(assert.AnError)

		_, err := repo.FindRoomIDsByUserID(20, []uint{10, 11})

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
