package roomuserfavourite

import (
	"testing"

	"sound-stage-backend/internal/pkg/testutil"

	"github.com/stretchr/testify/require"
)

type roomUserRow struct {
	RoomID    uint
	UserID    uint
	IsBlocked bool
}

func (roomUserRow) TableName() string { return "room_users" }

func TestRepo_Add_Integration(t *testing.T) {
	t.Run("persists a room user favourite", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &RoomUserFavourite{})
		repo := NewRepo(db)

		err := repo.Add(20, 10)
		require.NoError(t, err)

		var fav RoomUserFavourite
		require.NoError(t, db.First(&fav, "user_id = ? AND room_id = ?", 20, 10).Error)
		require.Equal(t, uint(20), fav.UserID)
		require.Equal(t, uint(10), fav.RoomID)
	})

	t.Run("is idempotent on duplicate favourites", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &RoomUserFavourite{})
		repo := NewRepo(db)

		require.NoError(t, repo.Add(20, 10))
		require.NoError(t, repo.Add(20, 10))

		var count int64
		require.NoError(t, db.Model(&RoomUserFavourite{}).Where("user_id = ? AND room_id = ?", 20, 10).Count(&count).Error)
		require.Equal(t, int64(1), count)
	})
}

func TestRepo_Remove_Integration(t *testing.T) {
	t.Run("removes an existing room user favourite", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &RoomUserFavourite{})
		repo := NewRepo(db)

		require.NoError(t, repo.Add(20, 10))
		require.NoError(t, repo.Remove(20, 10))

		var count int64
		require.NoError(t, db.Model(&RoomUserFavourite{}).Where("user_id = ? AND room_id = ?", 20, 10).Count(&count).Error)
		require.Equal(t, int64(0), count)
	})

	t.Run("succeeds when favourite does not exist", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &RoomUserFavourite{})
		repo := NewRepo(db)

		err := repo.Remove(20, 10)
		require.NoError(t, err)
	})
}
