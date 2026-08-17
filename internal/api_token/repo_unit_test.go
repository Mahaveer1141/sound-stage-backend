package apitoken

import (
	"database/sql/driver"
	"regexp"
	"testing"

	"sound-stage-backend/internal/pkg/testutil"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestRepo_FindByToken_Unit(t *testing.T) {
	cols := []string{"id", "token", "type", "user_id", "is_active"}

	t.Run("returns token when found and active", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)
		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "api_tokens" WHERE token = $1 AND is_active = $2 ORDER BY "api_tokens"."id" LIMIT $3`)).
			WithArgs(driver.Value("tok-123"), driver.Value(true), driver.Value(1)).
			WillReturnRows(sqlmock.NewRows(cols).AddRow(1, "tok-123", "access", 42, true))
		got, err := repo.FindByToken("tok-123")
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, uint(1), got.ID)
		assert.Equal(t, "tok-123", got.Token)
		assert.Equal(t, uint(42), got.UserID)
		assert.True(t, got.IsActive)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when not found", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "api_tokens" WHERE token = $1 AND is_active = $2 ORDER BY "api_tokens"."id" LIMIT $3`)).
			WithArgs(driver.Value("missing"), driver.Value(true), driver.Value(1)).
			WillReturnError(gorm.ErrRecordNotFound)

		got, err := repo.FindByToken("missing")

		require.Error(t, err)
		require.Nil(t, got)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepo_CreateToken_Unit(t *testing.T) {
	t.Run("creates and returns a token", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(
			`INSERT INTO "api_tokens" ("created_at","updated_at","token","type","is_active","user_id") VALUES ($1,$2,$3,$4,$5,$6) RETURNING "id"`)).
			WithArgs(
				sqlmock.AnyArg(),
				sqlmock.AnyArg(),
				driver.Value("tok-abc"),
				driver.Value(string(AccessToken)),
				driver.Value(true),
				driver.Value(uint(7)),
			).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
		mock.ExpectCommit()

		got, err := repo.CreateToken(CreateAPITokenInput{
			Token:  "tok-abc",
			Type:   AccessToken,
			UserID: 7,
		})

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, uint(1), got.ID)
		assert.Equal(t, "tok-abc", got.Token)
		assert.True(t, got.IsActive)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error on insert failure", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(
			`INSERT INTO "api_tokens" ("created_at","updated_at","token","type","is_active","user_id") VALUES ($1,$2,$3,$4,$5,$6) RETURNING "id"`)).
			WithArgs(
				sqlmock.AnyArg(),
				sqlmock.AnyArg(),
				driver.Value("tok-xyz"),
				driver.Value(string(AccessToken)),
				driver.Value(true),
				driver.Value(uint(7)),
			).
			WillReturnError(assert.AnError)
		mock.ExpectRollback()

		got, err := repo.CreateToken(CreateAPITokenInput{
			Token:  "tok-xyz",
			Type:   AccessToken,
			UserID: 7,
		})

		require.Error(t, err)
		require.Nil(t, got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepo_Deactivate_Unit(t *testing.T) {
	t.Run("deactivates active tokens for user", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "api_tokens" SET "is_active"=$1,"updated_at"=$2 WHERE user_id = $3 AND is_active = $4`)).
			WithArgs(driver.Value(false), sqlmock.AnyArg(), driver.Value(uint(7)), driver.Value(true)).
			WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectCommit()

		err := repo.Deactivate(7)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error on update failure", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "api_tokens" SET "is_active"=$1,"updated_at"=$2 WHERE user_id = $3 AND is_active = $4`)).
			WithArgs(driver.Value(false), sqlmock.AnyArg(), driver.Value(uint(7)), driver.Value(true)).
			WillReturnError(assert.AnError)
		mock.ExpectRollback()

		err := repo.Deactivate(7)

		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
