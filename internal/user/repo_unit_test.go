package user

import (
	"regexp"
	"testing"
	"time"

	"sound-stage-backend/internal/pkg/testutil"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestRepo_Create_Unit(t *testing.T) {
	t.Run("creates a user", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(
			`INSERT INTO "users" ("created_at","updated_at","email","first_name","last_name","last_login_at","deleted_at") VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING "id"`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "user@example.com", "First", "Last", nil, sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
		mock.ExpectCommit()

		got, err := repo.Create(&CreateUserParams{
			Email:     "user@example.com",
			FirstName: "First",
			LastName:  "Last",
		})

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, uint(1), got.ID)
		assert.Equal(t, "user@example.com", got.Email)
		assert.Equal(t, "First", got.FirstName)
		require.NotNil(t, got.LastName)
		assert.Equal(t, "Last", *got.LastName)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("creates a user without last name", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(
			`INSERT INTO "users" ("created_at","updated_at","email","first_name","last_name","last_login_at","deleted_at") VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING "id"`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "user@example.com", "First", "", nil, sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
		mock.ExpectCommit()

		got, err := repo.Create(&CreateUserParams{
			Email:     "user@example.com",
			FirstName: "First",
		})

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, "First", got.FirstName)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when insert fails", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(
			`INSERT INTO "users" ("created_at","updated_at","email","first_name","last_name","last_login_at","deleted_at") VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING "id"`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "user@example.com", "First", "Last", nil, sqlmock.AnyArg()).
			WillReturnError(assert.AnError)
		mock.ExpectRollback()

		got, err := repo.Create(&CreateUserParams{
			Email:     "user@example.com",
			FirstName: "First",
			LastName:  "Last",
		})

		require.Error(t, err)
		require.Nil(t, got)
		assert.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepo_FindByEmail_Unit(t *testing.T) {
	t.Run("finds a user by email", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`SELECT \* FROM "users" WHERE email = \$1 AND "users"\."deleted_at" IS NULL ORDER BY "users"\."id" LIMIT \$2`,
		).
			WithArgs("user@example.com", 1).
			WillReturnRows(
				sqlmock.NewRows([]string{"id", "created_at", "updated_at", "email", "first_name", "last_name", "last_login_at", "deleted_at"}).
					AddRow(1, time.Now(), time.Now(), "user@example.com", "First", "Last", nil, nil),
			)

		got, err := repo.FindByEmail("USER@example.com")

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, uint(1), got.ID)
		assert.Equal(t, "user@example.com", got.Email)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns nil when not found", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`SELECT \* FROM "users" WHERE email = \$1 AND "users"\."deleted_at" IS NULL ORDER BY "users"\."id" LIMIT \$2`,
		).
			WithArgs("missing@example.com", 1).
			WillReturnError(gorm.ErrRecordNotFound)

		got, err := repo.FindByEmail("missing@example.com")

		require.NoError(t, err)
		require.Nil(t, got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepo_FindByID_Unit(t *testing.T) {
	t.Run("finds a user by id", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`SELECT \* FROM "users" WHERE id = \$1 AND "users"\."deleted_at" IS NULL ORDER BY "users"\."id" LIMIT \$2`,
		).
			WithArgs(1, 1).
			WillReturnRows(
				sqlmock.NewRows([]string{"id", "created_at", "updated_at", "email", "first_name", "last_name", "last_login_at", "deleted_at"}).
					AddRow(1, time.Now(), time.Now(), "user@example.com", "First", "Last", nil, nil),
			)

		got, err := repo.FindByID(1)

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, uint(1), got.ID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns nil when not found", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`SELECT \* FROM "users" WHERE id = \$1 AND "users"\."deleted_at" IS NULL ORDER BY "users"\."id" LIMIT \$2`,
		).
			WithArgs(999, 1).
			WillReturnError(gorm.ErrRecordNotFound)

		got, err := repo.FindByID(999)

		require.NoError(t, err)
		require.Nil(t, got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepo_UpdateLastLoginAt_Unit(t *testing.T) {
	t.Run("updates last login at", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "users" SET "last_login_at"=$1,"updated_at"=$2 WHERE id = $3`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 1).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := repo.UpdateLastLoginAt(1)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when update fails", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "users" SET "last_login_at"=$1,"updated_at"=$2 WHERE id = $3`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 1).
			WillReturnError(assert.AnError)
		mock.ExpectRollback()

		err := repo.UpdateLastLoginAt(1)

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepo_Update_Unit(t *testing.T) {
	t.Run("updates first and last name", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`SELECT \* FROM "users" WHERE id = \$1 AND "users"\."deleted_at" IS NULL ORDER BY "users"\."id" LIMIT \$2`,
		).
			WithArgs(1, 1).
			WillReturnRows(
				sqlmock.NewRows([]string{"id", "created_at", "updated_at", "email", "first_name", "last_name", "last_login_at", "deleted_at"}).
					AddRow(1, time.Now(), time.Now(), "user@example.com", "First", "Last", nil, nil),
			)

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "users" SET "created_at"=$1,"updated_at"=$2,"email"=$3,"first_name"=$4,"last_name"=$5,"last_login_at"=$6,"deleted_at"=$7 WHERE "users"."deleted_at" IS NULL AND "id" = $8`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "user@example.com", "Updated", "New", nil, nil, 1).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		got, err := repo.Update(1, &UpdateUserParams{FirstName: "Updated", LastName: "New"})

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, "Updated", got.FirstName)
		require.NotNil(t, got.LastName)
		assert.Equal(t, "New", *got.LastName)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when user not found", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`SELECT \* FROM "users" WHERE id = \$1 AND "users"\."deleted_at" IS NULL ORDER BY "users"\."id" LIMIT \$2`,
		).
			WithArgs(999, 1).
			WillReturnError(gorm.ErrRecordNotFound)

		got, err := repo.Update(999, &UpdateUserParams{FirstName: "Updated", LastName: "Last"})

		require.Error(t, err)
		require.Nil(t, got)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
