package roomuserfavourite

import (
	user "sound-stage-backend/internal/user"
	"testing"

	"sound-stage-backend/internal/pkg/listopts"
	"sound-stage-backend/internal/pkg/testutil"
	"sound-stage-backend/internal/room"

	"github.com/stretchr/testify/require"
)

type roomUserBlock struct {
	RoomID uint
	UserID uint
}

func (roomUserBlock) TableName() string { return "room_user_blocks" }

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

func TestRepo_ListUserFavourites_Integration(t *testing.T) {
	t.Run("returns paginated favourite rooms for a user", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &RoomUserFavourite{}, &room.Room{}, &user.User{}, &roomUserBlock{})
		repo := NewRepo(db)

		creator := user.User{Email: "creator@example.com", FirstName: "Creator"}
		require.NoError(t, db.Create(&creator).Error)

		room1 := room.Room{Name: "Room One", Description: "First", Type: room.RoomTypePublic, CreatorID: creator.ID}
		room2 := room.Room{Name: "Room Two", Description: "Second", Type: room.RoomTypePublic, CreatorID: creator.ID}
		room3 := room.Room{Name: "Room Three", Description: "Third", Type: room.RoomTypePublic, CreatorID: creator.ID}
		require.NoError(t, db.Create(&room1).Error)
		require.NoError(t, db.Create(&room2).Error)
		require.NoError(t, db.Create(&room3).Error)

		require.NoError(t, repo.Add(20, room1.ID))
		require.NoError(t, repo.Add(20, room2.ID))
		require.NoError(t, repo.Add(30, room3.ID))
		require.NoError(t, db.Create(&roomUserBlock{RoomID: room2.ID, UserID: 20}).Error)

		favourites, err := repo.ListUserFavourites(20, listopts.Pagination{Page: 1, PageSize: 10})
		require.NoError(t, err)
		require.Len(t, favourites, 1)
		require.Equal(t, room1.ID, favourites[0].RoomID)
		require.Equal(t, "Room One", favourites[0].Room.Name)

		count, err := repo.CountByUserID(20)
		require.NoError(t, err)
		require.Equal(t, int64(1), count)
	})
}
