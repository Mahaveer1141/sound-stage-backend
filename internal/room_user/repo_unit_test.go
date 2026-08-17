package roomuser

import (
	"regexp"
	"testing"
	"time"

	"sound-stage-backend/internal/model"
	"sound-stage-backend/internal/pkg/listopts"
	"sound-stage-backend/internal/pkg/testutil"
	"sound-stage-backend/internal/role"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestRepo_Create_Unit(t *testing.T) {
	t.Run("creates a room user using the repo db when tx is nil", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(
			`INSERT INTO "room_users" ("created_at","updated_at","user_id","room_id","role_id","last_joined_at","last_left_at","is_online") VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING "id"`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 1, 10, 100, sqlmock.AnyArg(), sqlmock.AnyArg(), false).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
		mock.ExpectCommit()

		got, err := repo.Create(nil, 1, 10, 100)

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, uint(1), got.ID)
		assert.Equal(t, uint(1), got.UserID)
		assert.Equal(t, uint(10), got.RoomID)
		assert.Equal(t, uint(100), got.RoleID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("creates a room user with an explicit tx", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectBegin()
		tx := gdb.Begin()
		require.NoError(t, tx.Error)

		mock.ExpectQuery(regexp.QuoteMeta(
			`INSERT INTO "room_users" ("created_at","updated_at","user_id","room_id","role_id","last_joined_at","last_left_at","is_online") VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING "id"`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 2, 20, 200, sqlmock.AnyArg(), sqlmock.AnyArg(), false).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(2))

		got, err := repo.Create(tx, 2, 20, 200)

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, uint(2), got.ID)

		mock.ExpectCommit()
		require.NoError(t, tx.Commit().Error)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when insert fails", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(
			`INSERT INTO "room_users" ("created_at","updated_at","user_id","room_id","role_id","last_joined_at","last_left_at","is_online") VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING "id"`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 1, 10, 100, sqlmock.AnyArg(), sqlmock.AnyArg(), false).
			WillReturnError(assert.AnError)
		mock.ExpectRollback()

		got, err := repo.Create(nil, 1, 10, 100)

		require.Error(t, err)
		require.Nil(t, got)
		assert.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepo_FindBy_Unit(t *testing.T) {
	t.Run("returns room user when found", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`SELECT \* FROM "room_users" WHERE user_id = \$1 AND room_id = \$2 ORDER BY "room_users"\."id" LIMIT \$3`,
		).
			WithArgs(1, 10, 1).
			WillReturnRows(
				sqlmock.NewRows([]string{"id", "created_at", "updated_at", "user_id", "room_id", "role_id", "last_joined_at", "last_left_at", "is_online"}).
					AddRow(1, time.Now(), time.Now(), 1, 10, 100, time.Now(), time.Now(), true),
			)

		mock.ExpectQuery(`SELECT \* FROM "roles" WHERE "roles"\."id" = \$1.*`).
			WithArgs(100).
			WillReturnRows(
				sqlmock.NewRows([]string{"id", "created_at", "updated_at", "name", "description"}).
					AddRow(100, time.Now(), time.Now(), "listener", nil),
			)

		mock.ExpectQuery(`SELECT \* FROM "users" WHERE "users"\."id" = \$1.*`).
			WithArgs(1).
			WillReturnRows(
				sqlmock.NewRows([]string{"id", "created_at", "updated_at", "email", "first_name", "last_name", "last_login_at", "deleted_at"}).
					AddRow(1, time.Now(), time.Now(), "user@example.com", "U", "ser", nil, nil),
			)

		got, err := repo.FindBy(1, 10)

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, uint(1), got.ID)
		assert.Equal(t, uint(1), got.UserID)
		assert.Equal(t, "user@example.com", got.User.Email)
		assert.Equal(t, "listener", got.Role.Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns nil when not found", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`SELECT \* FROM "room_users" WHERE user_id = \$1 AND room_id = \$2 ORDER BY "room_users"\."id" LIMIT \$3`,
		).
			WithArgs(1, 10, 1).
			WillReturnError(gorm.ErrRecordNotFound)

		got, err := repo.FindBy(1, 10)

		require.NoError(t, err)
		require.Nil(t, got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns wrapped database error for other errors", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`SELECT \* FROM "room_users" WHERE user_id = \$1 AND room_id = \$2 ORDER BY "room_users"\."id" LIMIT \$3`,
		).
			WithArgs(1, 10, 1).
			WillReturnError(assert.AnError)

		got, err := repo.FindBy(1, 10)

		require.Error(t, err)
		require.Nil(t, got)
		assert.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepo_UpdateActivity_Unit(t *testing.T) {
	t.Run("updates join activity", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "room_users" SET "is_online"=$1,"last_joined_at"=$2,"updated_at"=$3 WHERE "id" = $4`)).
			WithArgs(true, sqlmock.AnyArg(), sqlmock.AnyArg(), 1).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := repo.UpdateActivity(&RoomUser{BaseModel: model.BaseModel{ID: 1}}, ActivityJoin)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("updates leave activity", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "room_users" SET "is_online"=$1,"last_left_at"=$2,"updated_at"=$3 WHERE "id" = $4`)).
			WithArgs(false, sqlmock.AnyArg(), sqlmock.AnyArg(), 2).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := repo.UpdateActivity(&RoomUser{BaseModel: model.BaseModel{ID: 2}}, ActivityLeave)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepo_HasRoles_Unit(t *testing.T) {
	t.Run("returns true when user has one of the roles", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`SELECT count\(\*\) FROM "room_users" JOIN roles ON roles\.id = room_users\.role_id WHERE room_users\.user_id = \$1 AND room_users\.room_id = \$2 AND roles\.name IN \(\$3\)`,
		).
			WithArgs(1, 10, "owner").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		got, err := repo.HasRoles(1, 10, []role.RoleName{role.RoleOwner})

		require.NoError(t, err)
		assert.True(t, got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns false when user has none of the roles", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`SELECT count\(\*\) FROM "room_users" JOIN roles ON roles\.id = room_users\.role_id WHERE room_users\.user_id = \$1 AND room_users\.room_id = \$2 AND roles\.name IN \(\$3\)`,
		).
			WithArgs(1, 10, "owner").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		got, err := repo.HasRoles(1, 10, []role.RoleName{role.RoleOwner})

		require.NoError(t, err)
		assert.False(t, got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when count query fails", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`SELECT count\(\*\) FROM "room_users" JOIN roles ON roles\.id = room_users\.role_id WHERE room_users\.user_id = \$1 AND room_users\.room_id = \$2 AND roles\.name IN \(\$3\)`,
		).
			WithArgs(1, 10, "owner").
			WillReturnError(assert.AnError)

		got, err := repo.HasRoles(1, 10, []role.RoleName{role.RoleOwner})

		require.Error(t, err)
		assert.False(t, got)
		assert.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepo_ListByRoomID_Unit(t *testing.T) {
	t.Run("returns room users with default sort and pagination", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`SELECT .*FROM "room_users" WHERE room_users\.room_id = \$1 ORDER BY room_users\.created_at desc LIMIT \$2`,
		).
			WithArgs(10, 10).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "user_id", "room_id", "role_id", "last_joined_at", "last_left_at", "is_online"}))

		got, err := repo.ListByRoomID(10, RoomUserFilter{}, listopts.Sort{}, listopts.Pagination{Page: 1, PageSize: 10})

		require.NoError(t, err)
		assert.Empty(t, got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("filters by role, sorts, and paginates with offset", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`SELECT .*FROM "room_users" JOIN roles ON roles\.id = room_users\.role_id WHERE room_users\.room_id = \$1 AND roles\.name IN \(\$2\) ORDER BY room_users\.created_at asc LIMIT \$3 OFFSET \$4`,
		).
			WithArgs(10, "listener", 2, 2).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "user_id", "room_id", "role_id", "last_joined_at", "last_left_at", "is_online"}))

		got, err := repo.ListByRoomID(
			10,
			RoomUserFilter{Roles: []string{"listener"}},
			listopts.Sort{Field: "created_at", Order: "asc"},
			listopts.Pagination{Page: 2, PageSize: 2},
		)

		require.NoError(t, err)
		assert.Empty(t, got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepo_CountByRoomID_Unit(t *testing.T) {
	t.Run("returns total count without filter", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`SELECT count\(\*\) FROM "room_users" WHERE room_users\.room_id = \$1`,
		).
			WithArgs(10).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

		got, err := repo.CountByRoomID(10, RoomUserFilter{})

		require.NoError(t, err)
		assert.Equal(t, int64(5), got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns count with role filter", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`SELECT count\(\*\) FROM "room_users" JOIN roles ON roles\.id = room_users\.role_id WHERE room_users\.room_id = \$1 AND roles\.name IN \(\$2\)`,
		).
			WithArgs(10, "listener").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

		got, err := repo.CountByRoomID(10, RoomUserFilter{Roles: []string{"listener"}})

		require.NoError(t, err)
		assert.Equal(t, int64(3), got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepo_UpdateRole_Unit(t *testing.T) {
	t.Run("updates role by id", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "room_users" SET "role_id"=$1 WHERE room_id = $2 AND user_id = $3`)).
			WithArgs(300, 10, 1).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := repo.UpdateRole(10, 1, 300)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when update fails", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "room_users" SET "role_id"=$1 WHERE room_id = $2 AND user_id = $3`)).
			WithArgs(300, 10, 1).
			WillReturnError(assert.AnError)
		mock.ExpectRollback()

		err := repo.UpdateRole(10, 1, 300)

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
