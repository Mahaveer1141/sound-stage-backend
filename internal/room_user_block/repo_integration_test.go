package roomuserblock

import (
	"testing"

	"sound-stage-backend/internal/pkg/listopts"
	"sound-stage-backend/internal/pkg/testutil"
	"sound-stage-backend/internal/user"

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

func TestRepo_ListByRoomID_Integration(t *testing.T) {
	t.Run("returns blocked users with pagination and preloaded users", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &RoomUserBlock{}, &user.User{})
		repo := NewRepo(db)

		actor := user.User{Email: "actor@example.com", FirstName: "Actor"}
		blocked1 := user.User{Email: "b1@example.com", FirstName: "B1"}
		blocked2 := user.User{Email: "b2@example.com", FirstName: "B2"}
		require.NoError(t, db.Create(&actor).Error)
		require.NoError(t, db.Create(&blocked1).Error)
		require.NoError(t, db.Create(&blocked2).Error)

		require.NoError(t, repo.Add(10, blocked1.ID, actor.ID))
		require.NoError(t, repo.Add(10, blocked2.ID, actor.ID))
		require.NoError(t, repo.Add(20, blocked1.ID, actor.ID))

		blocks, err := repo.ListByRoomID(10, listopts.Pagination{Page: 1, PageSize: 1})
		require.NoError(t, err)
		require.Len(t, blocks, 1)
		require.Equal(t, blocked2.ID, blocks[0].UserID)
		require.Equal(t, "B2", blocks[0].User.FirstName)
		require.Equal(t, "Actor", blocks[0].BlockedBy.FirstName)

		count, err := repo.CountByRoomID(10)
		require.NoError(t, err)
		require.Equal(t, int64(2), count)
	})
}
