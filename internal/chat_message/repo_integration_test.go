package chatmessage

import (
	"testing"

	"github.com/stretchr/testify/require"

	"sound-stage-backend/internal/pkg/listopts"
	"sound-stage-backend/internal/pkg/testutil"
	"sound-stage-backend/internal/room"
	"sound-stage-backend/internal/user"
)

func TestRepo_Create_Integration(t *testing.T) {
	t.Run("persists a chat message", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &user.User{}, &room.Room{}, &ChatMessage{})
		repo := NewRepo(db)

		u := user.User{Email: "user@example.com", FirstName: "Test"}
		require.NoError(t, db.Create(&u).Error)

		r := room.Room{Name: "Stage", CreatorID: u.ID, Type: room.RoomTypePublic}
		require.NoError(t, db.Create(&r).Error)

		got, err := repo.Create(&CreateChatMessageParams{
			RoomID:   r.ID,
			UserID:   u.ID,
			Content:  "hello",
			IsPinned: true,
		})

		require.NoError(t, err)
		require.NotNil(t, got)
		require.NotZero(t, got.ID)
		require.Equal(t, r.ID, got.RoomID)
		require.Equal(t, u.ID, got.UserID)
		require.Equal(t, "hello", got.Content)
		require.True(t, got.IsPinned)

		var fetched ChatMessage
		require.NoError(t, db.First(&fetched, got.ID).Error)
		require.Equal(t, "hello", fetched.Content)
		require.True(t, fetched.IsPinned)
	})
}

func TestRepo_List_Integration(t *testing.T) {
	t.Run("returns messages for a room with preloaded user", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &user.User{}, &room.Room{}, &ChatMessage{})
		repo := NewRepo(db)

		u := user.User{Email: "user@example.com", FirstName: "Test"}
		require.NoError(t, db.Create(&u).Error)

		r := room.Room{Name: "Stage", CreatorID: u.ID, Type: room.RoomTypePublic}
		require.NoError(t, db.Create(&r).Error)

		require.NoError(t, db.Create(&ChatMessage{RoomID: r.ID, UserID: u.ID, Content: "first"}).Error)
		require.NoError(t, db.Create(&ChatMessage{RoomID: r.ID, UserID: u.ID, Content: "second", IsPinned: true}).Error)

		got, err := repo.List(ChatMessageFilter{RoomID: r.ID}, listopts.Pagination{Page: 1, PageSize: 10})

		require.NoError(t, err)
		require.Len(t, got, 2)
		require.Equal(t, "second", got[0].Content)
		require.Equal(t, "first", got[1].Content)
		require.Equal(t, u.ID, got[0].User.ID)
	})

	t.Run("filters by isPinned", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &user.User{}, &room.Room{}, &ChatMessage{})
		repo := NewRepo(db)

		u := user.User{Email: "user@example.com", FirstName: "Test"}
		require.NoError(t, db.Create(&u).Error)

		r := room.Room{Name: "Stage", CreatorID: u.ID, Type: room.RoomTypePublic}
		require.NoError(t, db.Create(&r).Error)

		require.NoError(t, db.Create(&ChatMessage{RoomID: r.ID, UserID: u.ID, Content: "first"}).Error)
		require.NoError(t, db.Create(&ChatMessage{RoomID: r.ID, UserID: u.ID, Content: "second", IsPinned: true}).Error)

		isPinned := true
		got, err := repo.List(ChatMessageFilter{RoomID: r.ID, IsPinned: &isPinned}, listopts.Pagination{Page: 1, PageSize: 10})

		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, "second", got[0].Content)
	})
}

func TestRepo_Count_Integration(t *testing.T) {
	t.Run("returns count with filter", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &user.User{}, &room.Room{}, &ChatMessage{})
		repo := NewRepo(db)

		u := user.User{Email: "user@example.com", FirstName: "Test"}
		require.NoError(t, db.Create(&u).Error)

		r := room.Room{Name: "Stage", CreatorID: u.ID, Type: room.RoomTypePublic}
		require.NoError(t, db.Create(&r).Error)

		require.NoError(t, db.Create(&ChatMessage{RoomID: r.ID, UserID: u.ID, Content: "first"}).Error)
		require.NoError(t, db.Create(&ChatMessage{RoomID: r.ID, UserID: u.ID, Content: "second", IsPinned: true}).Error)

		isPinned := true
		got, err := repo.Count(ChatMessageFilter{RoomID: r.ID, IsPinned: &isPinned})

		require.NoError(t, err)
		require.Equal(t, int64(1), got)
	})
}
