package chatmessage

import (
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"sound-stage-backend/internal/pkg/listopts"
	"sound-stage-backend/internal/pkg/testutil"
)

func TestRepo_Create_Unit(t *testing.T) {
	now := time.Now()
	userCols := []string{"id", "created_at", "updated_at", "email", "first_name", "last_name", "last_login_at"}

	t.Run("creates a chat message", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(
			`INSERT INTO "chat_messages" ("created_at","updated_at","room_id","user_id","content","is_pinned") VALUES ($1,$2,$3,$4,$5,$6) RETURNING "id"`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 1, 2, "hello", false).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
		mock.ExpectCommit()

		mock.ExpectQuery(
			`SELECT \* FROM "chat_messages" WHERE "chat_messages"\."id" = \$1 AND "chat_messages"\."id" = \$2 ORDER BY "chat_messages"\."id" LIMIT \$3`,
		).
			WithArgs(1, 1, 1).
			WillReturnRows(
				sqlmock.NewRows([]string{"id", "created_at", "updated_at", "room_id", "user_id", "content", "is_pinned"}).
					AddRow(1, now, now, 1, 2, "hello", false),
			)

		mock.ExpectQuery(
			`SELECT \* FROM "users" WHERE "users"\."id" = \$1`,
		).
			WithArgs(2).
			WillReturnRows(
				sqlmock.NewRows(userCols).
					AddRow(2, now, now, "user@example.com", "Test", "User", nil),
			)

		got, err := repo.Create(&CreateChatMessageParams{RoomID: 1, UserID: 2, Content: "hello"})

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, uint(1), got.ID)
		assert.Equal(t, uint(1), got.RoomID)
		assert.Equal(t, uint(2), got.UserID)
		assert.Equal(t, "hello", got.Content)
		assert.False(t, got.IsPinned)
		assert.Equal(t, uint(2), got.User.ID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when insert fails", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(
			`INSERT INTO "chat_messages" ("created_at","updated_at","room_id","user_id","content","is_pinned") VALUES ($1,$2,$3,$4,$5,$6) RETURNING "id"`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 1, 2, "hello", false).
			WillReturnError(assert.AnError)
		mock.ExpectRollback()

		got, err := repo.Create(&CreateChatMessageParams{RoomID: 1, UserID: 2, Content: "hello"})

		require.Error(t, err)
		require.Nil(t, got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepo_List_Unit(t *testing.T) {
	now := time.Now()
	userCols := []string{"id", "created_at", "updated_at", "email", "first_name", "last_name", "last_login_at"}

	t.Run("returns messages with preloaded user", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`SELECT \* FROM "chat_messages" WHERE chat_messages\.room_id = \$1 ORDER BY chat_messages\.created_at desc LIMIT \$2`,
		).
			WithArgs(1, 10).
			WillReturnRows(
				sqlmock.NewRows([]string{"id", "created_at", "updated_at", "room_id", "user_id", "content", "is_pinned"}).
					AddRow(1, now, now, 1, 2, "hello", false),
			)

		mock.ExpectQuery(
			`SELECT \* FROM "users" WHERE "users"\."id" = \$1`,
		).
			WithArgs(2).
			WillReturnRows(
				sqlmock.NewRows(userCols).
					AddRow(2, now, now, "user@example.com", "Test", "User", nil),
			)

		got, err := repo.List(ChatMessageFilter{RoomID: 1}, listopts.Pagination{Page: 1, PageSize: 10})

		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "hello", got[0].Content)
		assert.Equal(t, uint(2), got[0].User.ID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when query fails", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`SELECT \* FROM "chat_messages" WHERE chat_messages\.room_id = \$1 ORDER BY chat_messages\.created_at desc LIMIT \$2`,
		).
			WithArgs(1, 10).
			WillReturnError(assert.AnError)

		got, err := repo.List(ChatMessageFilter{RoomID: 1}, listopts.Pagination{Page: 1, PageSize: 10})

		require.Error(t, err)
		require.Nil(t, got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepo_Count_Unit(t *testing.T) {
	t.Run("returns total count for a room", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`SELECT count\(\*\) FROM "chat_messages" WHERE chat_messages\.room_id = \$1`,
		).
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

		got, err := repo.Count(ChatMessageFilter{RoomID: 1})

		require.NoError(t, err)
		assert.Equal(t, int64(5), got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when count fails", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`SELECT count\(\*\) FROM "chat_messages" WHERE chat_messages\.room_id = \$1`,
		).
			WithArgs(1).
			WillReturnError(assert.AnError)

		got, err := repo.Count(ChatMessageFilter{RoomID: 1})

		require.Error(t, err)
		assert.Zero(t, got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
