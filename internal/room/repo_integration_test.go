package room

import (
	"testing"

	"sound-stage-backend/internal/pkg/listopts"
	"sound-stage-backend/internal/pkg/testutil"
	"sound-stage-backend/internal/user"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestRepo_Create_Integration(t *testing.T) {
	t.Run("persists a new room and returns it", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &Room{}, &user.User{})
		repo := NewRepo(db)

		u := user.User{Email: "creator@example.com", FirstName: "Creator"}
		require.NoError(t, db.Create(&u).Error)

		got, err := repo.Create(db, &CreateRoomParams{
			Name:        "Stage A",
			Description: "The main stage",
			CreatorID:   u.ID,
		})

		require.NoError(t, err)
		require.NotNil(t, got)
		require.NotZero(t, got.ID)
		require.Equal(t, "Stage A", got.Name)
		require.Equal(t, "The main stage", got.Description)
		require.Equal(t, u.ID, got.CreatorID)

		var fetched Room
		require.NoError(t, db.First(&fetched, got.ID).Error)
		require.Equal(t, "Stage A", fetched.Name)
	})
}

func TestRepo_List_Integration(t *testing.T) {
	t.Run("returns rooms sorted and paginated with preloaded creator", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &Room{}, &user.User{})
		repo := NewRepo(db)

		c1 := user.User{Email: "c1@example.com", FirstName: "C1"}
		c2 := user.User{Email: "c2@example.com", FirstName: "C2"}
		require.NoError(t, db.Create(&c1).Error)
		require.NoError(t, db.Create(&c2).Error)

		require.NoError(t, db.Create(&Room{Name: "Beta Room", Description: "B", CreatorID: c1.ID}).Error)
		require.NoError(t, db.Create(&Room{Name: "Alpha Room", Description: "A", CreatorID: c2.ID}).Error)
		require.NoError(t, db.Create(&Room{Name: "Gamma Room", Description: "G", CreatorID: c1.ID}).Error)

		got, err := repo.List(
			RoomFilter{},
			listopts.Sort{Field: "name", Order: "asc"},
			listopts.Pagination{Page: 1, PageSize: 2},
		)

		require.NoError(t, err)
		require.Len(t, got, 2)
		require.Equal(t, "Alpha Room", got[0].Name)
		require.Equal(t, "Beta Room", got[1].Name)
		require.Equal(t, c2.ID, got[0].Creator.ID)
		require.Equal(t, c1.ID, got[1].Creator.ID)
	})

	t.Run("filters rooms by name query", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &Room{}, &user.User{})
		repo := NewRepo(db)

		c := user.User{Email: "creator@example.com", FirstName: "Creator"}
		require.NoError(t, db.Create(&c).Error)

		require.NoError(t, db.Create(&Room{Name: "Alpha Room", Description: "A", CreatorID: c.ID}).Error)
		require.NoError(t, db.Create(&Room{Name: "Beta Hall", Description: "B", CreatorID: c.ID}).Error)
		require.NoError(t, db.Create(&Room{Name: "Another Room", Description: "C", CreatorID: c.ID}).Error)

		got, err := repo.List(
			RoomFilter{Query: "room"},
			listopts.Sort{Field: "name", Order: "asc"},
			listopts.Pagination{Page: 1, PageSize: 10},
		)

		require.NoError(t, err)
		require.Len(t, got, 2)
		require.Equal(t, "Alpha Room", got[0].Name)
		require.Equal(t, "Another Room", got[1].Name)
	})
}

func TestRepo_Count_Integration(t *testing.T) {
	t.Run("returns total count without filter", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &Room{}, &user.User{})
		repo := NewRepo(db)

		c := user.User{Email: "creator@example.com", FirstName: "Creator"}
		require.NoError(t, db.Create(&c).Error)

		require.NoError(t, db.Create(&Room{Name: "Room One", Description: "1", CreatorID: c.ID}).Error)
		require.NoError(t, db.Create(&Room{Name: "Room Two", Description: "2", CreatorID: c.ID}).Error)

		got, err := repo.Count(RoomFilter{})
		require.NoError(t, err)
		require.Equal(t, int64(2), got)
	})

	t.Run("returns count for matching filter", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &Room{}, &user.User{})
		repo := NewRepo(db)

		c := user.User{Email: "creator@example.com", FirstName: "Creator"}
		require.NoError(t, db.Create(&c).Error)

		require.NoError(t, db.Create(&Room{Name: "Room One", Description: "1", CreatorID: c.ID}).Error)
		require.NoError(t, db.Create(&Room{Name: "Hall Two", Description: "2", CreatorID: c.ID}).Error)

		got, err := repo.Count(RoomFilter{Query: "room"})
		require.NoError(t, err)
		require.Equal(t, int64(1), got)
	})
}

func TestRepo_FindByID_Integration(t *testing.T) {
	t.Run("finds a room with creator and users preloaded", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &Room{}, &user.User{})
		repo := NewRepo(db)

		c := user.User{Email: "creator@example.com", FirstName: "Creator"}
		require.NoError(t, db.Create(&c).Error)

		room := Room{Name: "Stage A", Description: "Desc", CreatorID: c.ID}
		require.NoError(t, db.Create(&room).Error)

		got, err := repo.FindByID(room.ID)

		require.NoError(t, err)
		require.NotNil(t, got)
		require.Equal(t, room.ID, got.ID)
		require.Equal(t, "Stage A", got.Name)
		require.Equal(t, c.ID, got.Creator.ID)
		require.Empty(t, got.Users)
	})

	t.Run("returns error when room does not exist", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &Room{}, &user.User{})
		repo := NewRepo(db)

		got, err := repo.FindByID(999)

		require.Error(t, err)
		require.Nil(t, got)
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})
}

func TestRepo_Update_Integration(t *testing.T) {
	t.Run("updates a room's name and description", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &Room{}, &user.User{})
		repo := NewRepo(db)

		c := user.User{Email: "creator@example.com", FirstName: "Creator"}
		require.NoError(t, db.Create(&c).Error)

		room := Room{Name: "Old Name", Description: "Old", CreatorID: c.ID}
		require.NoError(t, db.Create(&room).Error)

		got, err := repo.Update(room.ID, &UpdateRoomParams{
			Name:        "New Name",
			Description: "New",
		})

		require.NoError(t, err)
		require.NotNil(t, got)
		require.Equal(t, "New Name", got.Name)
		require.Equal(t, "New", got.Description)

		var fetched Room
		require.NoError(t, db.First(&fetched, room.ID).Error)
		require.Equal(t, "New Name", fetched.Name)
		require.Equal(t, "New", fetched.Description)
	})

	t.Run("returns error for non-existent room", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &Room{}, &user.User{})
		repo := NewRepo(db)

		got, err := repo.Update(999, &UpdateRoomParams{Name: "Name", Description: "Desc"})

		require.Error(t, err)
		require.Nil(t, got)
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})
}
