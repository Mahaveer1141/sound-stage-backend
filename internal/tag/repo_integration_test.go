package tag

import (
	"testing"

	"sound-stage-backend/internal/pkg/listopts"
	"sound-stage-backend/internal/pkg/testutil"

	"github.com/stretchr/testify/require"
)

func TestRepo_Create_Integration(t *testing.T) {
	t.Run("persists a tag with just a name", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &Tag{})
		repo := NewRepo(db)

		got, err := repo.Create(&CreateTagParams{Name: "live"})

		require.NoError(t, err)
		require.NotNil(t, got)
		require.NotZero(t, got.ID)
		require.Equal(t, "live", got.Name)

		var fetched Tag
		require.NoError(t, db.First(&fetched, got.ID).Error)
		require.Equal(t, got.ID, fetched.ID)
		require.Equal(t, "live", fetched.Name)
	})

	t.Run("does not allow the same tag name twice", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &Tag{})
		repo := NewRepo(db)

		_, err := repo.Create(&CreateTagParams{Name: "live"})
		require.NoError(t, err)

		_, err = repo.Create(&CreateTagParams{Name: "live"})
		require.Error(t, err)
	})
}

func TestRepo_List_Integration(t *testing.T) {
	t.Run("lists tags with search and pagination", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &Tag{})
		repo := NewRepo(db)

		_, err := repo.Create(&CreateTagParams{Name: "live"})
		require.NoError(t, err)
		_, err = repo.Create(&CreateTagParams{Name: "jazz"})
		require.NoError(t, err)
		_, err = repo.Create(&CreateTagParams{Name: "news"})
		require.NoError(t, err)

		got, err := repo.List(
			TagFilter{},
			listopts.Sort{Field: "name", Order: "asc"},
			listopts.Pagination{Page: 1, PageSize: 10},
		)

		require.NoError(t, err)
		require.Len(t, got, 3)

		count, err := repo.Count(TagFilter{})
		require.NoError(t, err)
		require.Equal(t, int64(3), count)
	})

	t.Run("searches tags by name", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &Tag{})
		repo := NewRepo(db)

		_, err := repo.Create(&CreateTagParams{Name: "live"})
		require.NoError(t, err)
		_, err = repo.Create(&CreateTagParams{Name: "jazz"})
		require.NoError(t, err)

		got, err := repo.List(
			TagFilter{Query: "ja"},
			listopts.Sort{Field: "name", Order: "asc"},
			listopts.Pagination{Page: 1, PageSize: 10},
		)

		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, "jazz", got[0].Name)

		count, err := repo.Count(TagFilter{Query: "ja"})
		require.NoError(t, err)
		require.Equal(t, int64(1), count)
	})

	t.Run("respects pagination", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &Tag{})
		repo := NewRepo(db)

		_, err := repo.Create(&CreateTagParams{Name: "a"})
		require.NoError(t, err)
		_, err = repo.Create(&CreateTagParams{Name: "b"})
		require.NoError(t, err)

		got, err := repo.List(
			TagFilter{},
			listopts.Sort{Field: "name", Order: "asc"},
			listopts.Pagination{Page: 2, PageSize: 1},
		)

		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, "b", got[0].Name)
	})
}
