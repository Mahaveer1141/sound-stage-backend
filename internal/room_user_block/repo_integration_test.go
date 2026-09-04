package roomuserblock

import (
	"testing"

	"sound-stage-backend/internal/pkg/testutil"

	"github.com/stretchr/testify/require"
)

func TestRepo_Add_Integration(t *testing.T) {
	t.Run("persists a room user block", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &RoomUserBlock{})
		repo := NewRepo(db)

		err := repo.Add(10, 20, 30)
		require.NoError(t, err)

		var block RoomUserBlock
		require.NoError(t, db.First(&block, "room_id = ? AND user_id = ? AND blocked_by_id = ?", 10, 20, 30).Error)
		require.Equal(t, uint(10), block.RoomID)
		require.Equal(t, uint(20), block.UserID)
		require.Equal(t, uint(30), block.BlockedByID)
	})

	t.Run("is idempotent on duplicate blocks", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &RoomUserBlock{})
		repo := NewRepo(db)

		require.NoError(t, repo.Add(10, 20, 30))
		require.NoError(t, repo.Add(10, 20, 30))

		var count int64
		require.NoError(t, db.Model(&RoomUserBlock{}).Where("room_id = ? AND user_id = ?", 10, 20).Count(&count).Error)
		require.Equal(t, int64(1), count)
	})
}

func TestRepo_Remove_Integration(t *testing.T) {
	t.Run("removes an existing room user block", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &RoomUserBlock{})
		repo := NewRepo(db)

		require.NoError(t, repo.Add(10, 20, 30))
		require.NoError(t, repo.Remove(10, 20))

		var count int64
		require.NoError(t, db.Model(&RoomUserBlock{}).Where("room_id = ? AND user_id = ?", 10, 20).Count(&count).Error)
		require.Equal(t, int64(0), count)
	})

	t.Run("succeeds when block does not exist", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &RoomUserBlock{})
		repo := NewRepo(db)

		err := repo.Remove(10, 20)
		require.NoError(t, err)
	})
}
