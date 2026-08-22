package tag

import (
	"regexp"
	"testing"

	"sound-stage-backend/internal/pkg/listopts"
	"sound-stage-backend/internal/pkg/testutil"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepo_Create_Unit(t *testing.T) {
	insertSQL := `INSERT INTO "tags" ("created_at","updated_at","name") VALUES ($1,$2,$3) RETURNING "id"`

	t.Run("persists a tag and returns it", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(insertSQL)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "live").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
		mock.ExpectCommit()

		got, err := repo.Create(&CreateTagParams{Name: "live"})

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, uint(1), got.ID)
		assert.Equal(t, "live", got.Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("trims surrounding whitespace from the name", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(insertSQL)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "live").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(2))
		mock.ExpectCommit()

		got, err := repo.Create(&CreateTagParams{Name: "  live  "})

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, "live", got.Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when insert fails", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(insertSQL)).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "live").
			WillReturnError(assert.AnError)
		mock.ExpectRollback()

		got, err := repo.Create(&CreateTagParams{Name: "live"})

		require.Error(t, err)
		require.Nil(t, got)
		assert.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepo_List_Unit(t *testing.T) {
	t.Run("returns empty list with default sort and pagination", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "tags" ORDER BY tags.created_at desc LIMIT $1`)).
			WithArgs(10).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name"}))

		got, err := repo.List(
			TagFilter{},
			listopts.Sort{},
			listopts.Pagination{Page: 1, PageSize: 10},
		)

		require.NoError(t, err)
		assert.Empty(t, got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns tags with search filter and explicit sort", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "tags" WHERE tags.name LIKE $1 ORDER BY tags.name asc LIMIT $2`)).
			WithArgs("%ja%", 10).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "jazz"))

		got, err := repo.List(
			TagFilter{Query: "ja"},
			listopts.Sort{Field: "name", Order: "asc"},
			listopts.Pagination{Page: 1, PageSize: 10},
		)

		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "jazz", got[0].Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("respects offset for later pages", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "tags" ORDER BY tags.created_at desc LIMIT $1 OFFSET $2`)).
			WithArgs(1, 1).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(2, "news"))

		got, err := repo.List(
			TagFilter{},
			listopts.Sort{},
			listopts.Pagination{Page: 2, PageSize: 1},
		)

		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "news", got[0].Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when query fails", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "tags" ORDER BY tags.created_at desc LIMIT $1`)).
			WithArgs(10).
			WillReturnError(assert.AnError)

		got, err := repo.List(
			TagFilter{},
			listopts.Sort{},
			listopts.Pagination{Page: 1, PageSize: 10},
		)

		require.Error(t, err)
		assert.Empty(t, got)
		assert.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepo_Count_Unit(t *testing.T) {
	t.Run("returns total count without filter", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "tags"`)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

		got, err := repo.Count(TagFilter{})

		require.NoError(t, err)
		assert.Equal(t, int64(3), got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns count with search filter", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT count(*) FROM "tags" WHERE tags.name LIKE $1`)).
			WithArgs("%ja%").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		got, err := repo.Count(TagFilter{Query: "ja"})

		require.NoError(t, err)
		assert.Equal(t, int64(1), got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when count query fails", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "tags"`)).
			WillReturnError(assert.AnError)

		got, err := repo.Count(TagFilter{})

		require.Error(t, err)
		require.Zero(t, got)
		assert.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
