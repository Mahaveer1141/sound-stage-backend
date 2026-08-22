package category

import (
	"testing"

	"sound-stage-backend/internal/pkg/testutil"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func seedCategories(t *testing.T, db *gorm.DB) []Category {
	t.Helper()
	names := []string{"Gaming", "Music & Audio", "Tech", "Entertainment", "Learning", "Business & Career"}
	categories := make([]Category, 0, len(names))
	for _, name := range names {
		c := Category{Name: name}
		require.NoError(t, db.Create(&c).Error)
		categories = append(categories, c)
	}
	return categories
}

func TestRepo_List_Integration(t *testing.T) {
	t.Run("lists all seeded categories", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &Category{})
		repo := NewRepo(db)

		seedCategories(t, db)

		got, err := repo.List()

		require.NoError(t, err)
		require.Len(t, got, 6)
	})
}
