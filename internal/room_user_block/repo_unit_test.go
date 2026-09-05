package roomuserblock

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
	t.Run("creates a room user block", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(
			`INSERT INTO "room_user_blocks" ("created_at","updated_at","user_id","room_id","blocked_by_id") VALUES ($1,$2,$3,$4,$5) ON CONFLICT DO NOTHING RETURNING "id"`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 20, 10, 30).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
		mock.ExpectCommit()

		err := repo.Add(10, 20, 30)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when insert fails", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(
			`INSERT INTO "room_user_blocks" ("created_at","updated_at","user_id","room_id","blocked_by_id") VALUES ($1,$2,$3,$4,$5) ON CONFLICT DO NOTHING RETURNING "id"`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 20, 10, 30).
			WillReturnError(assert.AnError)
		mock.ExpectRollback()

		err := repo.Add(10, 20, 30)

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepo_Remove_Unit(t *testing.T) {
	t.Run("deletes a room user block", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			`DELETE FROM "room_user_blocks" WHERE room_id = $1 AND user_id = $2`)).
			WithArgs(10, 20).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := repo.Remove(10, 20)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when delete fails", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			`DELETE FROM "room_user_blocks" WHERE room_id = $1 AND user_id = $2`)).
			WithArgs(10, 20).
			WillReturnError(assert.AnError)
		mock.ExpectRollback()

		err := repo.Remove(10, 20)

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepo_ListByRoomID_Unit(t *testing.T) {
	t.Run("returns error when query fails", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`SELECT \* FROM "room_user_blocks" WHERE room_id = \$1 ORDER BY id DESC LIMIT \$2`,
		).
			WithArgs(10, 10).
			WillReturnError(assert.AnError)

		_, err := repo.ListByRoomID(10, listopts.Pagination{Page: 1, PageSize: 10})

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepo_CountByRoomID_Unit(t *testing.T) {
	t.Run("returns the number of blocked users in the room", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`SELECT count\(\*\) FROM "room_user_blocks" WHERE room_id = \$1`,
		).
			WithArgs(10).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

		got, err := repo.CountByRoomID(10)

		require.NoError(t, err)
		assert.Equal(t, int64(5), got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when count query fails", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`SELECT count\(\*\) FROM "room_user_blocks" WHERE room_id = \$1`,
		).
			WithArgs(10).
			WillReturnError(assert.AnError)

		_, err := repo.CountByRoomID(10)

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
