package roomuserfavourite

import (
	"regexp"
	"testing"

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
