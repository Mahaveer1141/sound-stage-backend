package room

import (
	"testing"

	"sound-stage-backend/internal/category"
	"sound-stage-backend/internal/pkg/listopts"
	"sound-stage-backend/internal/pkg/testutil"
	roomcategory "sound-stage-backend/internal/room_category"
	"sound-stage-backend/internal/tag"
	"sound-stage-backend/internal/tagging"
	"sound-stage-backend/internal/user"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type roomUserRow struct {
	RoomID    uint
	UserID    uint
	IsBlocked bool
}

func (roomUserRow) TableName() string { return "room_users" }

func TestRepo_Create_Integration(t *testing.T) {
	t.Run("persists a new room and returns it", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &user.User{}, &category.Category{}, &roomcategory.RoomCategory{}, &tag.Tag{}, &tagging.Tagging{}, &Room{})
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

	t.Run("creates a room with categories and tags", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &user.User{}, &category.Category{}, &roomcategory.RoomCategory{}, &tag.Tag{}, &tagging.Tagging{}, &Room{})
		repo := NewRepo(db)

		u := user.User{Email: "creator2@example.com", FirstName: "Creator"}
		require.NoError(t, db.Create(&u).Error)

		c1 := category.Category{Name: "Concert"}
		c2 := category.Category{Name: "Podcast"}
		require.NoError(t, db.Create(&c1).Error)
		require.NoError(t, db.Create(&c2).Error)

		t1 := tag.Tag{Name: "jazz"}
		t2 := tag.Tag{Name: "live"}
		require.NoError(t, db.Create(&t1).Error)
		require.NoError(t, db.Create(&t2).Error)

		got, err := repo.Create(db, &CreateRoomParams{
			Name:        "Tagged Room",
			Description: "With tags",
			CreatorID:   u.ID,
			CategoryIds: []uint{c1.ID, c2.ID},
			TagIds:      []uint{t1.ID, t2.ID},
		})

		require.NoError(t, err)
		require.NotNil(t, got)
		require.NotZero(t, got.ID)

		var roomCategories []roomcategory.RoomCategory
		require.NoError(t, db.Where("room_id = ?", got.ID).Find(&roomCategories).Error)
		require.Len(t, roomCategories, 2)

		var taggings []tagging.Tagging
		require.NoError(t, db.Where("taggable_type = ? AND taggable_id = ?", "Room", got.ID).Find(&taggings).Error)
		require.Len(t, taggings, 2)
	})
}

func TestRepo_List_Integration(t *testing.T) {
	t.Run("returns rooms sorted and paginated with preloaded creator", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &user.User{}, &category.Category{}, &roomcategory.RoomCategory{}, &tag.Tag{}, &tagging.Tagging{}, &Room{})
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
		db := testutil.NewIntegrationDB(t, &user.User{}, &category.Category{}, &roomcategory.RoomCategory{}, &tag.Tag{}, &tagging.Tagging{}, &Room{})
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

	t.Run("filters rooms by category ids", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &user.User{}, &category.Category{}, &roomcategory.RoomCategory{}, &tag.Tag{}, &tagging.Tagging{}, &Room{})
		repo := NewRepo(db)

		c := user.User{Email: "creator@example.com", FirstName: "Creator"}
		require.NoError(t, db.Create(&c).Error)

		cat1 := category.Category{Name: "Concert"}
		cat2 := category.Category{Name: "Podcast"}
		require.NoError(t, db.Create(&cat1).Error)
		require.NoError(t, db.Create(&cat2).Error)

		room1 := Room{Name: "Room One", CreatorID: c.ID}
		room2 := Room{Name: "Room Two", CreatorID: c.ID}
		room3 := Room{Name: "Room Three", CreatorID: c.ID}
		require.NoError(t, db.Create(&room1).Error)
		require.NoError(t, db.Create(&room2).Error)
		require.NoError(t, db.Create(&room3).Error)

		require.NoError(t, db.Create(&roomcategory.RoomCategory{RoomID: room1.ID, CategoryID: cat1.ID}).Error)
		require.NoError(t, db.Create(&roomcategory.RoomCategory{RoomID: room2.ID, CategoryID: cat2.ID}).Error)

		got, err := repo.List(
			RoomFilter{CategoryIds: []uint{cat1.ID}},
			listopts.Sort{Field: "name", Order: "asc"},
			listopts.Pagination{Page: 1, PageSize: 10},
		)

		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, "Room One", got[0].Name)
	})

	t.Run("filters rooms by tag ids", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &user.User{}, &category.Category{}, &roomcategory.RoomCategory{}, &tag.Tag{}, &tagging.Tagging{}, &Room{})
		repo := NewRepo(db)

		c := user.User{Email: "creator@example.com", FirstName: "Creator"}
		require.NoError(t, db.Create(&c).Error)

		tag1 := tag.Tag{Name: "jazz"}
		tag2 := tag.Tag{Name: "rock"}
		require.NoError(t, db.Create(&tag1).Error)
		require.NoError(t, db.Create(&tag2).Error)

		room1 := Room{Name: "Room One", CreatorID: c.ID}
		room2 := Room{Name: "Room Two", CreatorID: c.ID}
		room3 := Room{Name: "Room Three", CreatorID: c.ID}
		require.NoError(t, db.Create(&room1).Error)
		require.NoError(t, db.Create(&room2).Error)
		require.NoError(t, db.Create(&room3).Error)

		require.NoError(t, db.Create(&tagging.Tagging{TagID: tag1.ID, TaggableID: room1.ID, TaggableType: "Room"}).Error)
		require.NoError(t, db.Create(&tagging.Tagging{TagID: tag2.ID, TaggableID: room2.ID, TaggableType: "Room"}).Error)

		got, err := repo.List(
			RoomFilter{TagIds: []uint{tag1.ID}},
			listopts.Sort{Field: "name", Order: "asc"},
			listopts.Pagination{Page: 1, PageSize: 10},
		)

		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, "Room One", got[0].Name)
	})

	t.Run("filters rooms by type", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &user.User{}, &category.Category{}, &roomcategory.RoomCategory{}, &tag.Tag{}, &tagging.Tagging{}, &Room{})
		repo := NewRepo(db)

		c := user.User{Email: "creator@example.com", FirstName: "Creator"}
		require.NoError(t, db.Create(&c).Error)

		require.NoError(t, db.Create(&Room{Name: "Public Room", CreatorID: c.ID, Type: RoomTypePublic}).Error)
		require.NoError(t, db.Create(&Room{Name: "Private Room", CreatorID: c.ID, Type: RoomTypePrivate}).Error)
		require.NoError(t, db.Create(&Room{Name: "Another Public", CreatorID: c.ID, Type: RoomTypePublic}).Error)

		roomType := RoomTypePrivate
		got, err := repo.List(
			RoomFilter{Type: &roomType},
			listopts.Sort{Field: "name", Order: "asc"},
			listopts.Pagination{Page: 1, PageSize: 10},
		)

		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, "Private Room", got[0].Name)
	})

	t.Run("excludes rooms where the user is blocked", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &user.User{}, &category.Category{}, &roomcategory.RoomCategory{}, &tag.Tag{}, &tagging.Tagging{}, &Room{}, &roomUserRow{})
		repo := NewRepo(db)

		c := user.User{Email: "creator@example.com", FirstName: "Creator"}
		blockedUser := user.User{Email: "blocked@example.com", FirstName: "Blocked"}
		require.NoError(t, db.Create(&c).Error)
		require.NoError(t, db.Create(&blockedUser).Error)

		r1 := Room{Name: "Allowed Room", Description: "A", CreatorID: c.ID}
		r2 := Room{Name: "Blocked Room", Description: "B", CreatorID: c.ID}
		require.NoError(t, db.Create(&r1).Error)
		require.NoError(t, db.Create(&r2).Error)

		require.NoError(t, db.Create(&roomUserRow{RoomID: r2.ID, UserID: blockedUser.ID, IsBlocked: true}).Error)

		got, err := repo.List(
			RoomFilter{UserID: blockedUser.ID},
			listopts.Sort{Field: "name", Order: "asc"},
			listopts.Pagination{Page: 1, PageSize: 10},
		)

		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, "Allowed Room", got[0].Name)
	})
}

func TestRepo_Count_Integration(t *testing.T) {
	t.Run("returns total count without filter", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &user.User{}, &category.Category{}, &roomcategory.RoomCategory{}, &tag.Tag{}, &tagging.Tagging{}, &Room{})
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
		db := testutil.NewIntegrationDB(t, &user.User{}, &category.Category{}, &roomcategory.RoomCategory{}, &tag.Tag{}, &tagging.Tagging{}, &Room{})
		repo := NewRepo(db)

		c := user.User{Email: "creator@example.com", FirstName: "Creator"}
		require.NoError(t, db.Create(&c).Error)

		require.NoError(t, db.Create(&Room{Name: "Room One", Description: "1", CreatorID: c.ID}).Error)
		require.NoError(t, db.Create(&Room{Name: "Hall Two", Description: "2", CreatorID: c.ID}).Error)

		got, err := repo.Count(RoomFilter{Query: "room"})
		require.NoError(t, err)
		require.Equal(t, int64(1), got)
	})

	t.Run("returns count for category filter", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &user.User{}, &category.Category{}, &roomcategory.RoomCategory{}, &tag.Tag{}, &tagging.Tagging{}, &Room{})
		repo := NewRepo(db)

		c := user.User{Email: "creator@example.com", FirstName: "Creator"}
		require.NoError(t, db.Create(&c).Error)

		cat := category.Category{Name: "Concert"}
		require.NoError(t, db.Create(&cat).Error)

		room1 := Room{Name: "Room One", CreatorID: c.ID}
		room2 := Room{Name: "Room Two", CreatorID: c.ID}
		require.NoError(t, db.Create(&room1).Error)
		require.NoError(t, db.Create(&room2).Error)

		require.NoError(t, db.Create(&roomcategory.RoomCategory{RoomID: room1.ID, CategoryID: cat.ID}).Error)

		got, err := repo.Count(RoomFilter{CategoryIds: []uint{cat.ID}})
		require.NoError(t, err)
		require.Equal(t, int64(1), got)
	})

	t.Run("returns count for tag filter", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &user.User{}, &category.Category{}, &roomcategory.RoomCategory{}, &tag.Tag{}, &tagging.Tagging{}, &Room{})
		repo := NewRepo(db)

		c := user.User{Email: "creator@example.com", FirstName: "Creator"}
		require.NoError(t, db.Create(&c).Error)

		tg := tag.Tag{Name: "jazz"}
		require.NoError(t, db.Create(&tg).Error)

		room1 := Room{Name: "Room One", CreatorID: c.ID}
		room2 := Room{Name: "Room Two", CreatorID: c.ID}
		require.NoError(t, db.Create(&room1).Error)
		require.NoError(t, db.Create(&room2).Error)

		require.NoError(t, db.Create(&tagging.Tagging{TagID: tg.ID, TaggableID: room1.ID, TaggableType: "Room"}).Error)

		got, err := repo.Count(RoomFilter{TagIds: []uint{tg.ID}})
		require.NoError(t, err)
		require.Equal(t, int64(1), got)
	})

	t.Run("returns count for type filter", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &user.User{}, &category.Category{}, &roomcategory.RoomCategory{}, &tag.Tag{}, &tagging.Tagging{}, &Room{})
		repo := NewRepo(db)

		c := user.User{Email: "creator@example.com", FirstName: "Creator"}
		require.NoError(t, db.Create(&c).Error)

		require.NoError(t, db.Create(&Room{Name: "Public One", CreatorID: c.ID, Type: RoomTypePublic}).Error)
		require.NoError(t, db.Create(&Room{Name: "Public Two", CreatorID: c.ID, Type: RoomTypePublic}).Error)
		require.NoError(t, db.Create(&Room{Name: "Private One", CreatorID: c.ID, Type: RoomTypePrivate}).Error)

		roomType := RoomTypePrivate
		got, err := repo.Count(RoomFilter{Type: &roomType})
		require.NoError(t, err)
		require.Equal(t, int64(1), got)
	})
}

func TestRepo_FindByID_Integration(t *testing.T) {
	t.Run("finds a room with creator and users preloaded", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &user.User{}, &category.Category{}, &roomcategory.RoomCategory{}, &tag.Tag{}, &tagging.Tagging{}, &Room{})
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
		db := testutil.NewIntegrationDB(t, &user.User{}, &category.Category{}, &roomcategory.RoomCategory{}, &tag.Tag{}, &tagging.Tagging{}, &Room{})
		repo := NewRepo(db)

		got, err := repo.FindByID(999)

		require.Error(t, err)
		require.Nil(t, got)
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})
}

func TestRepo_Update_Integration(t *testing.T) {
	t.Run("updates a room's name and description", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &user.User{}, &category.Category{}, &roomcategory.RoomCategory{}, &tag.Tag{}, &tagging.Tagging{}, &Room{})
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
		db := testutil.NewIntegrationDB(t, &user.User{}, &category.Category{}, &roomcategory.RoomCategory{}, &tag.Tag{}, &tagging.Tagging{}, &Room{})
		repo := NewRepo(db)

		got, err := repo.Update(999, &UpdateRoomParams{Name: "Name", Description: "Desc"})

		require.Error(t, err)
		require.Nil(t, got)
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("replaces categories and tags", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &user.User{}, &category.Category{}, &roomcategory.RoomCategory{}, &tag.Tag{}, &tagging.Tagging{}, &Room{})
		repo := NewRepo(db)

		u := user.User{Email: "creator3@example.com", FirstName: "Creator"}
		require.NoError(t, db.Create(&u).Error)

		oldCat := category.Category{Name: "Old Category"}
		newCat := category.Category{Name: "New Category"}
		require.NoError(t, db.Create(&oldCat).Error)
		require.NoError(t, db.Create(&newCat).Error)

		oldTag := tag.Tag{Name: "old-tag"}
		newTag := tag.Tag{Name: "new-tag"}
		require.NoError(t, db.Create(&oldTag).Error)
		require.NoError(t, db.Create(&newTag).Error)

		room := Room{Name: "Room", CreatorID: u.ID}
		require.NoError(t, db.Create(&room).Error)

		require.NoError(t, db.Create(&roomcategory.RoomCategory{RoomID: room.ID, CategoryID: oldCat.ID}).Error)
		require.NoError(t, db.Create(&tagging.Tagging{TagID: oldTag.ID, TaggableID: room.ID, TaggableType: "Room"}).Error)

		got, err := repo.Update(room.ID, &UpdateRoomParams{
			Name:        "Updated",
			Description: "Updated",
			CategoryIds: []uint{newCat.ID},
			TagIds:      []uint{newTag.ID},
		})

		require.NoError(t, err)
		require.NotNil(t, got)

		var roomCategories []roomcategory.RoomCategory
		require.NoError(t, db.Where("room_id = ?", room.ID).Find(&roomCategories).Error)
		require.Len(t, roomCategories, 1)
		require.Equal(t, newCat.ID, roomCategories[0].CategoryID)

		var taggings []tagging.Tagging
		require.NoError(t, db.Where("taggable_type = ? AND taggable_id = ?", "Room", room.ID).Find(&taggings).Error)
		require.Len(t, taggings, 1)
		require.Equal(t, newTag.ID, taggings[0].TagID)
	})
}

func TestRepo_LoadTagsForRooms_Integration(t *testing.T) {
	t.Run("returns tags grouped by room id, ignoring other taggable types", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &user.User{}, &category.Category{}, &roomcategory.RoomCategory{}, &tag.Tag{}, &tagging.Tagging{}, &Room{})
		repo := NewRepo(db)

		c := user.User{Email: "creator@example.com", FirstName: "Creator"}
		require.NoError(t, db.Create(&c).Error)

		room1 := Room{Name: "Room One", CreatorID: c.ID}
		room2 := Room{Name: "Room Two", CreatorID: c.ID}
		require.NoError(t, db.Create(&room1).Error)
		require.NoError(t, db.Create(&room2).Error)

		jazz := tag.Tag{Name: "jazz"}
		live := tag.Tag{Name: "live"}
		rock := tag.Tag{Name: "rock"}
		require.NoError(t, db.Create(&jazz).Error)
		require.NoError(t, db.Create(&live).Error)
		require.NoError(t, db.Create(&rock).Error)

		require.NoError(t, db.Create(&tagging.Tagging{TagID: jazz.ID, TaggableID: room1.ID, TaggableType: "Room"}).Error)
		require.NoError(t, db.Create(&tagging.Tagging{TagID: live.ID, TaggableID: room1.ID, TaggableType: "Room"}).Error)
		require.NoError(t, db.Create(&tagging.Tagging{TagID: rock.ID, TaggableID: room2.ID, TaggableType: "Room"}).Error)
		require.NoError(t, db.Create(&tagging.Tagging{TagID: rock.ID, TaggableID: c.ID, TaggableType: "User"}).Error)

		got, err := repo.LoadTagsForRooms([]uint{room1.ID, room2.ID})

		require.NoError(t, err)
		require.Len(t, got, 2)
		require.Len(t, got[room1.ID], 2)
		require.Len(t, got[room2.ID], 1)

		names := []string{got[room1.ID][0].Name, got[room1.ID][1].Name}
		require.ElementsMatch(t, []string{"jazz", "live"}, names)
		require.Equal(t, "rock", got[room2.ID][0].Name)
	})

	t.Run("returns empty map when rooms have no tags", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &user.User{}, &category.Category{}, &roomcategory.RoomCategory{}, &tag.Tag{}, &tagging.Tagging{}, &Room{})
		repo := NewRepo(db)

		c := user.User{Email: "creator@example.com", FirstName: "Creator"}
		require.NoError(t, db.Create(&c).Error)

		room := Room{Name: "Untagged Room", CreatorID: c.ID}
		require.NoError(t, db.Create(&room).Error)

		got, err := repo.LoadTagsForRooms([]uint{room.ID})

		require.NoError(t, err)
		require.Empty(t, got)
	})
}

func TestRepo_UpdatePrivateCode_Integration(t *testing.T) {
	newRoom := func(t *testing.T, db *gorm.DB, name string, privateCode *string) Room {
		t.Helper()
		c := user.User{Email: name + "@example.com", FirstName: "Creator"}
		require.NoError(t, db.Create(&c).Error)

		room := Room{Name: name, Description: "A room", CreatorID: c.ID, Type: RoomTypePrivate, PrivateCode: privateCode}
		require.NoError(t, db.Create(&room).Error)
		return room
	}

	t.Run("replaces an existing private code", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &user.User{}, &category.Category{}, &roomcategory.RoomCategory{}, &tag.Tag{}, &tagging.Tagging{}, &Room{})
		repo := NewRepo(db)

		oldCode := "stale-code"
		room := newRoom(t, db, "Rotating Room", &oldCode)

		require.NoError(t, repo.UpdatePrivateCode(room.ID, "fresh-code"))

		var fetched Room
		require.NoError(t, db.First(&fetched, room.ID).Error)
		require.NotNil(t, fetched.PrivateCode)
		require.Equal(t, "fresh-code", *fetched.PrivateCode)
	})

	t.Run("returns error when room not found", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &user.User{}, &category.Category{}, &roomcategory.RoomCategory{}, &tag.Tag{}, &tagging.Tagging{}, &Room{})
		repo := NewRepo(db)

		err := repo.UpdatePrivateCode(9999, "secret-123")

		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})
}
