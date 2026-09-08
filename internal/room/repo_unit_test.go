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
			`INSERT INTO "rooms" ("created_at","updated_at","name","description","creator_id","type","private_code","is_chat_enabled","deleted_at") VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING "id"`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "Room A", "Description A", 1, "public", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
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
			`INSERT INTO "rooms" ("created_at","updated_at","name","description","creator_id","type","private_code","is_chat_enabled","deleted_at") VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING "id"`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "Room B", "", 2, "public", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
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
			`INSERT INTO "rooms" ("created_at","updated_at","name","description","creator_id","type","private_code","is_chat_enabled","deleted_at") VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING "id"`)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "Room A", "Description A", 1, "public", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
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
		mock.MatchExpectationsInOrder(false)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`SELECT \* FROM "rooms" WHERE .*rooms\.name LIKE \$1.*ORDER BY rooms\.name asc LIMIT \$2`,
		).
			WithArgs("%foo%", 5).
			WillReturnRows(
				sqlmock.NewRows([]string{"id", "created_at", "updated_at", "name", "description", "creator_id"}).
					AddRow(1, time.Now(), time.Now(), "Foo Room", "", 1),
			)

		mock.ExpectQuery(`SELECT \* FROM "room_categories" WHERE "room_categories"\."room_id" = \$1.*`).
			WithArgs(1).
			WillReturnRows(
				sqlmock.NewRows([]string{"id", "created_at", "updated_at", "room_id", "category_id"}),
			)

		mock.ExpectQuery(`SELECT \* FROM "users" WHERE "users"\."id" = \$1.*`).
			WithArgs(1).
			WillReturnRows(
				sqlmock.NewRows([]string{"id", "created_at", "updated_at", "email", "first_name", "last_name", "last_login_at", "deleted_at"}),
			)

		mock.ExpectQuery(`SELECT \* FROM "file_attachments" WHERE "owner_type" = \$1 AND "file_attachments"\."owner_id" = \$2 AND context = \$3`).
			WithArgs("rooms", 1, "room_cover").
			WillReturnRows(
				sqlmock.NewRows([]string{"id", "created_at", "updated_at", "owner_type", "owner_id", "context", "public_id", "url", "resource_type", "bytes", "format", "width", "height"}),
			)

		mock.ExpectQuery(`SELECT \* FROM "file_attachments" WHERE "owner_type" = \$1 AND "file_attachments"\."owner_id" = \$2 AND context = \$3`).
			WithArgs("rooms", 1, "room_logo").
			WillReturnRows(
				sqlmock.NewRows([]string{"id", "created_at", "updated_at", "owner_type", "owner_id", "context", "public_id", "url", "resource_type", "bytes", "format", "width", "height"}),
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

	t.Run("filters rooms by category ids", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`SELECT .* FROM "rooms" JOIN room_categories ON room_categories\.room_id = rooms\.id WHERE room_categories\.category_id IN \(\$1,\$2\) AND "rooms"\."deleted_at" IS NULL ORDER BY rooms\.created_at desc LIMIT \$3`,
		).
			WithArgs(uint(1), uint(2), 10).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "name", "description", "creator_id"}))

		got, err := repo.List(
			RoomFilter{CategoryIds: []uint{1, 2}},
			listopts.Sort{},
			listopts.Pagination{Page: 1, PageSize: 10},
		)

		require.NoError(t, err)
		assert.Empty(t, got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("filters rooms by tag ids", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`SELECT .* FROM "rooms" JOIN taggables ON taggables\.taggable_type = \$1 AND taggables\.taggable_id = rooms\.id WHERE taggables\.tag_id IN \(\$2,\$3\) AND "rooms"\."deleted_at" IS NULL ORDER BY rooms\.created_at desc LIMIT \$4`,
		).
			WithArgs("rooms", uint(3), uint(4), 10).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "name", "description", "creator_id"}))

		got, err := repo.List(
			RoomFilter{TagIds: []uint{3, 4}},
			listopts.Sort{},
			listopts.Pagination{Page: 1, PageSize: 10},
		)

		require.NoError(t, err)
		assert.Empty(t, got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("filters rooms by type", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		roomType := RoomTypePrivate
		mock.ExpectQuery(
			`SELECT \* FROM "rooms" WHERE type = \$1 AND "rooms"\."deleted_at" IS NULL ORDER BY rooms\.created_at desc LIMIT \$2`,
		).
			WithArgs(roomType, 10).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "name", "description", "creator_id"}))

		got, err := repo.List(
			RoomFilter{Type: &roomType},
			listopts.Sort{},
			listopts.Pagination{Page: 1, PageSize: 10},
		)

		require.NoError(t, err)
		assert.Empty(t, got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("excludes rooms where the user is blocked", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`SELECT \* FROM "rooms" WHERE .*NOT EXISTS \(SELECT 1 FROM room_users WHERE room_users\.room_id = rooms\.id AND room_users\.user_id = \$1 AND room_users\.is_blocked = \$2\).*ORDER BY rooms\.created_at desc LIMIT \$3`,
		).
			WithArgs(42, true, 10).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "name", "description", "creator_id"}))

		got, err := repo.List(
			RoomFilter{UserID: 42},
			listopts.Sort{},
			listopts.Pagination{Page: 1, PageSize: 10},
		)

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
			`SELECT count\(\*\) FROM "rooms" WHERE .*rooms\.name LIKE \$1.*`,
		).
			WithArgs("%foo%").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

		got, err := repo.Count(RoomFilter{Query: "foo"})

		require.NoError(t, err)
		assert.Equal(t, int64(5), got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns count with category filter", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`SELECT count\(\*\) FROM "rooms" JOIN room_categories ON room_categories\.room_id = rooms\.id WHERE room_categories\.category_id IN \(\$1,\$2\) AND "rooms"\."deleted_at" IS NULL`,
		).
			WithArgs(uint(1), uint(2)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

		got, err := repo.Count(RoomFilter{CategoryIds: []uint{1, 2}})

		require.NoError(t, err)
		assert.Equal(t, int64(3), got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns count with tag filter", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`SELECT count\(\*\) FROM "rooms" JOIN taggables ON taggables\.taggable_type = \$1 AND taggables\.taggable_id = rooms\.id WHERE taggables\.tag_id IN \(\$2,\$3\) AND "rooms"\."deleted_at" IS NULL`,
		).
			WithArgs("rooms", uint(3), uint(4)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

		got, err := repo.Count(RoomFilter{TagIds: []uint{3, 4}})

		require.NoError(t, err)
		assert.Equal(t, int64(2), got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns count with type filter", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		roomType := RoomTypePrivate
		mock.ExpectQuery(
			`SELECT count\(\*\) FROM "rooms" WHERE type = \$1 AND "rooms"\."deleted_at" IS NULL`,
		).
			WithArgs(roomType).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(7))

		got, err := repo.Count(RoomFilter{Type: &roomType})

		require.NoError(t, err)
		assert.Equal(t, int64(7), got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("excludes count for rooms where the user is blocked", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`SELECT count\(\*\) FROM "rooms" WHERE .*NOT EXISTS \(SELECT 1 FROM room_users WHERE room_users\.room_id = rooms\.id AND room_users\.user_id = \$1 AND room_users\.is_blocked = \$2\).*`,
		).
			WithArgs(42, true).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

		got, err := repo.Count(RoomFilter{UserID: 42})

		require.NoError(t, err)
		assert.Equal(t, int64(3), got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepo_FindByID_Unit(t *testing.T) {
	t.Run("returns error when room not found", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`(?s)SELECT \* FROM "rooms" WHERE "rooms"\."id" = \$1 AND "rooms"\."deleted_at" IS NULL ORDER BY "rooms"\."id" LIMIT \$2`,
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
			`(?s)SELECT \* FROM "rooms" WHERE "rooms"\."id" = \$1 AND "rooms"\."deleted_at" IS NULL ORDER BY "rooms"\."id" LIMIT \$2`,
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
			`(?s)SELECT \* FROM "rooms" WHERE "rooms"\."id" = \$1 AND "rooms"\."deleted_at" IS NULL ORDER BY "rooms"\."id" LIMIT \$2`,
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

func TestRepo_LoadTagsForRooms_Unit(t *testing.T) {
	t.Run("returns tags grouped by room id", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		now := time.Now()

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT tags.*, taggables.taggable_id as room_id FROM "tags" JOIN taggables ON taggables.tag_id = tags.id WHERE taggables.taggable_type = $1 AND taggables.taggable_id IN ($2,$3)`)).
			WithArgs("rooms", uint(1), uint(2)).
			WillReturnRows(
				sqlmock.NewRows([]string{"id", "created_at", "updated_at", "name", "room_id"}).
					AddRow(1, now, now, "jazz", 1).
					AddRow(2, now, now, "live", 1).
					AddRow(3, now, now, "rock", 2),
			)

		got, err := repo.LoadTagsForRooms([]uint{1, 2})

		require.NoError(t, err)
		require.Len(t, got, 2)
		require.Len(t, got[1], 2)
		require.Len(t, got[2], 1)
		assert.Equal(t, "jazz", got[1][0].Name)
		assert.Equal(t, "live", got[1][1].Name)
		assert.Equal(t, "rock", got[2][0].Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns empty map when no rooms have tags", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT tags.*, taggables.taggable_id as room_id FROM "tags" JOIN taggables ON taggables.tag_id = tags.id WHERE taggables.taggable_type = $1 AND taggables.taggable_id IN ($2)`)).
			WithArgs("rooms", uint(1)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "name", "room_id"}))

		got, err := repo.LoadTagsForRooms([]uint{1})

		require.NoError(t, err)
		assert.Empty(t, got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when query fails", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT tags.*, taggables.taggable_id as room_id FROM "tags" JOIN taggables ON taggables.tag_id = tags.id WHERE taggables.taggable_type = $1 AND taggables.taggable_id IN ($2)`)).
			WithArgs("rooms", uint(1)).
			WillReturnError(assert.AnError)

		got, err := repo.LoadTagsForRooms([]uint{1})

		require.Error(t, err)
		require.Nil(t, got)
		assert.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepo_UpdatePrivateCode_Unit(t *testing.T) {
	const selectRoomSQL = `SELECT * FROM "rooms" WHERE "rooms"."id" = $1 AND "rooms"."deleted_at" IS NULL ORDER BY "rooms"."id" LIMIT $2`
	const updateRoomSQL = `UPDATE "rooms" SET "created_at"=$1,"updated_at"=$2,"name"=$3,"description"=$4,"creator_id"=$5,"type"=$6,"private_code"=$7,"is_chat_enabled"=$8,"deleted_at"=$9 WHERE "rooms"."deleted_at" IS NULL AND "id" = $10`

	expectRoomFound := func(mock sqlmock.Sqlmock, id uint64) {
		mock.ExpectQuery(regexp.QuoteMeta(selectRoomSQL)).
			WithArgs(id, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "name", "description", "creator_id", "type", "private_code", "is_chat_enabled", "deleted_at"}).
				AddRow(id, time.Now(), time.Now(), "Room A", "Description A", 5, "private", "old-code", true, nil))
	}

	t.Run("updates the private code of an existing room", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		expectRoomFound(mock, 1)

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(updateRoomSQL)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "Room A", "Description A", 5, "private", "new-code", sqlmock.AnyArg(), sqlmock.AnyArg(), 1).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := repo.UpdatePrivateCode(1, "new-code")

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when room not found", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(regexp.QuoteMeta(selectRoomSQL)).
			WithArgs(999, 1).
			WillReturnError(gorm.ErrRecordNotFound)

		err := repo.UpdatePrivateCode(999, "new-code")

		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when save fails", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		expectRoomFound(mock, 1)

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(updateRoomSQL)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "Room A", "Description A", 5, "private", "new-code", sqlmock.AnyArg(), sqlmock.AnyArg(), 1).
			WillReturnError(assert.AnError)
		mock.ExpectRollback()

		err := repo.UpdatePrivateCode(1, "new-code")

		require.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
