package room

import (
	"regexp"
	"testing"
	"time"

	"sound-stage-backend/internal/pkg/listopts"
	"sound-stage-backend/internal/pkg/testutil"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestRepo_Create_Unit(t *testing.T) {
	t.Run("creates a room with all fields", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectBegin()
		tx := gdb.Begin()
		require.NoError(t, tx.Error)

		mock.ExpectQuery(regexp.QuoteMeta(
			`INSERT INTO "rooms" ("created_at","updated_at","name","description","creator_id","deleted_at") VALUES ($1,$2,$3,$4,$5,$6) RETURNING "id"`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "Room A", "Description A", 1, sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

		got, err := repo.Create(tx, &CreateRoomParams{
			Name:        "Room A",
			Description: "Description A",
			CreatorID:   1,
		})

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, uint(1), got.ID)
		assert.Equal(t, "Room A", got.Name)
		assert.Equal(t, "Description A", got.Description)
		assert.Equal(t, uint(1), got.CreatorID)

		mock.ExpectCommit()
		require.NoError(t, tx.Commit().Error)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("creates a room without description", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectBegin()
		tx := gdb.Begin()
		require.NoError(t, tx.Error)

		mock.ExpectQuery(regexp.QuoteMeta(
			`INSERT INTO "rooms" ("created_at","updated_at","name","description","creator_id","deleted_at") VALUES ($1,$2,$3,$4,$5,$6) RETURNING "id"`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "Room B", "", 2, sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(2))

		got, err := repo.Create(tx, &CreateRoomParams{
			Name:      "Room B",
			CreatorID: 2,
		})

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, uint(2), got.ID)
		assert.Equal(t, "Room B", got.Name)
		assert.Equal(t, "", got.Description)

		mock.ExpectCommit()
		require.NoError(t, tx.Commit().Error)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when insert fails", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectBegin()
		tx := gdb.Begin()
		require.NoError(t, tx.Error)

		mock.ExpectQuery(regexp.QuoteMeta(
			`INSERT INTO "rooms" ("created_at","updated_at","name","description","creator_id","deleted_at") VALUES ($1,$2,$3,$4,$5,$6) RETURNING "id"`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "Room A", "Description A", 1, sqlmock.AnyArg()).
			WillReturnError(assert.AnError)

		got, err := repo.Create(tx, &CreateRoomParams{
			Name:        "Room A",
			Description: "Description A",
			CreatorID:   1,
		})

		require.Error(t, err)
		require.Nil(t, got)

		mock.ExpectRollback()
		require.NoError(t, tx.Rollback().Error)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepo_List_Unit(t *testing.T) {
	t.Run("returns empty list with default sort and pagination", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`SELECT \* FROM "rooms" WHERE "rooms"\."deleted_at" IS NULL ORDER BY rooms\.created_at desc LIMIT \$1`,
		).
			WithArgs(10).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "name", "description", "creator_id"}))

		got, err := repo.List(RoomFilter{}, listopts.Sort{}, listopts.Pagination{Page: 1, PageSize: 10})

		require.NoError(t, err)
		assert.Empty(t, got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("applies search filter and sort", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`SELECT \* FROM "rooms" WHERE .*LOWER\(rooms\.name\) LIKE LOWER\(\$1\).*ORDER BY rooms\.name asc LIMIT \$2`,
		).
			WithArgs("%foo%", 5).
			WillReturnRows(
				sqlmock.NewRows([]string{"id", "created_at", "updated_at", "name", "description", "creator_id"}).
					AddRow(1, time.Now(), time.Now(), "Foo Room", "", 1),
			)

		mock.ExpectQuery(`SELECT \* FROM "users" WHERE "users"\."id" = \$1.*`).
			WithArgs(1).
			WillReturnRows(
				sqlmock.NewRows([]string{"id", "created_at", "updated_at", "email", "first_name", "last_name", "last_login_at", "deleted_at"}),
			)

		got, err := repo.List(
			RoomFilter{Query: "foo"},
			listopts.Sort{Field: "name", Order: "asc"},
			listopts.Pagination{Page: 1, PageSize: 5},
		)

		require.NoError(t, err)
		assert.Len(t, got, 1)
		assert.Equal(t, "Foo Room", got[0].Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns correct page with offset and limit", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`SELECT \* FROM "rooms" WHERE "rooms"\."deleted_at" IS NULL ORDER BY rooms\.created_at desc LIMIT \$1 OFFSET \$2`,
		).
			WithArgs(2, 2).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "name", "description", "creator_id"}))

		got, err := repo.List(RoomFilter{}, listopts.Sort{}, listopts.Pagination{Page: 2, PageSize: 2})

		require.NoError(t, err)
		assert.Empty(t, got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepo_Count_Unit(t *testing.T) {
	t.Run("returns total count without filter", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`SELECT count\(\*\) FROM "rooms" WHERE "rooms"\."deleted_at" IS NULL`,
		).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(42))

		got, err := repo.Count(RoomFilter{})

		require.NoError(t, err)
		assert.Equal(t, int64(42), got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns count with search filter", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`SELECT count\(\*\) FROM "rooms" WHERE .*LOWER\(rooms\.name\) LIKE LOWER\(\$1\).*`,
		).
			WithArgs("%foo%").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

		got, err := repo.Count(RoomFilter{Query: "foo"})

		require.NoError(t, err)
		assert.Equal(t, int64(5), got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepo_FindByID_Unit(t *testing.T) {
	t.Run("returns error when room not found", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`(?s)SELECT \* FROM "rooms" WHERE id = \$1 AND "rooms"\."deleted_at" IS NULL ORDER BY "rooms"\."id" LIMIT \$2`,
		).
			WithArgs(1, 1).
			WillReturnError(gorm.ErrRecordNotFound)

		got, err := repo.FindByID(1)

		require.Error(t, err)
		require.Nil(t, got)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns wrapped database error for other errors", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`(?s)SELECT \* FROM "rooms" WHERE id = \$1 AND "rooms"\."deleted_at" IS NULL ORDER BY "rooms"\."id" LIMIT \$2`,
		).
			WithArgs(1, 1).
			WillReturnError(assert.AnError)

		got, err := repo.FindByID(1)

		require.Error(t, err)
		require.Nil(t, got)
		assert.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepo_Update_Unit(t *testing.T) {
	t.Run("returns error when room not found", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`(?s)SELECT \* FROM "rooms" WHERE id = \$1 AND "rooms"\."deleted_at" IS NULL ORDER BY "rooms"\."id" LIMIT \$2`,
		).
			WithArgs(1, 1).
			WillReturnError(gorm.ErrRecordNotFound)

		got, err := repo.Update(1, &UpdateRoomParams{Name: "Updated", Description: "Updated"})

		require.Error(t, err)
		require.Nil(t, got)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
