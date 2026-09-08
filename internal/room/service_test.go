package room

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	fileattachment "sound-stage-backend/internal/file_attachment"
	"sound-stage-backend/internal/pkg/httpx"
	"sound-stage-backend/internal/pkg/listopts"
	"sound-stage-backend/internal/pkg/testutil"
	"sound-stage-backend/internal/role"
	roomuser "sound-stage-backend/internal/room_user"
	"sound-stage-backend/internal/tag"
)

type mockRepository struct{ mock.Mock }

func (m *mockRepository) Create(tx *gorm.DB, input *CreateRoomParams) (*Room, error) {
	args := m.Called(tx, input)
	r, _ := args.Get(0).(*Room)
	return r, args.Error(1)
}
func (m *mockRepository) Update(id uint, input *UpdateRoomParams) (*Room, error) {
	args := m.Called(id, input)
	r, _ := args.Get(0).(*Room)
	return r, args.Error(1)
}
func (m *mockRepository) UpdatePrivateCode(id uint, code string) error {
	args := m.Called(id, code)
	return args.Error(0)
}
func (m *mockRepository) FindByID(id uint) (*Room, error) {
	args := m.Called(id)
	r, _ := args.Get(0).(*Room)
	return r, args.Error(1)
}
func (m *mockRepository) List(filter RoomFilter, sort listopts.Sort, p listopts.Pagination) ([]Room, error) {
	args := m.Called(filter, sort, p)
	rooms, _ := args.Get(0).([]Room)
	return rooms, args.Error(1)
}
func (m *mockRepository) Count(filter RoomFilter) (int64, error) {
	args := m.Called(filter)
	return args.Get(0).(int64), args.Error(1)
}
func (m *mockRepository) LoadTagsForRooms(roomIds []uint) (map[uint][]tag.Tag, error) {
	args := m.Called(roomIds)
	res, _ := args.Get(0).(map[uint][]tag.Tag)
	return res, args.Error(1)
}

type mockRoomUserService struct{ mock.Mock }

func (m *mockRoomUserService) Create(tx *gorm.DB, userID, roomID uint, roleName role.RoleName) (*roomuser.RoomUser, error) {
	args := m.Called(tx, userID, roomID, roleName)
	ru, _ := args.Get(0).(*roomuser.RoomUser)
	return ru, args.Error(1)
}

func (m *mockRoomUserService) Rejoin(ru *roomuser.RoomUser) error {
	args := m.Called(ru)
	return args.Error(0)
}
func (m *mockRoomUserService) RemoveUser(ctx context.Context, userID, roomID uint) error {
	args := m.Called(ctx, userID, roomID)
	return args.Error(0)
}
func (m *mockRoomUserService) FindBy(userID, roomID uint) (*roomuser.RoomUser, error) {
	args := m.Called(userID, roomID)
	ru, _ := args.Get(0).(*roomuser.RoomUser)
	return ru, args.Error(1)
}
func (m *mockRoomUserService) HasRoles(userID, roomID uint, permissions []role.RoleName) (bool, error) {
	args := m.Called(userID, roomID, permissions)
	return args.Bool(0), args.Error(1)
}
func (m *mockRoomUserService) MapByUserAndRoomIDs(userID uint, roomIDs []uint) (map[uint]*roomuser.RoomUser, error) {
	args := m.Called(userID, roomIDs)
	res, _ := args.Get(0).(map[uint]*roomuser.RoomUser)
	return res, args.Error(1)
}
func (m *mockRoomUserService) CountByRoomIDs(roomIDs []uint, filter roomuser.RoomUserFilter) (map[uint]int64, error) {
	args := m.Called(roomIDs, filter)
	res, _ := args.Get(0).(map[uint]int64)
	return res, args.Error(1)
}
func (m *mockRoomUserService) IsBlocked(roomID, userID uint) (bool, error) {
	args := m.Called(roomID, userID)
	return args.Bool(0), args.Error(1)
}
func (m *mockRoomUserService) SetMuted(ctx context.Context, roomID, userID, actorID uint, isMuted bool) error {
	args := m.Called(ctx, roomID, userID, actorID, isMuted)
	return args.Error(0)
}
func (m *mockRoomUserService) SetHandRaised(ctx context.Context, roomID, userID uint, isHandRaised bool) error {
	args := m.Called(ctx, roomID, userID, isHandRaised)
	return args.Error(0)
}

type mockFavouriteService struct{ mock.Mock }

func (m *mockFavouriteService) FavouritedRoomIDs(userID uint, roomIDs []uint) (map[uint]bool, error) {
	args := m.Called(userID, roomIDs)
	res, _ := args.Get(0).(map[uint]bool)
	return res, args.Error(1)
}

type mockFileAttachmentService struct{ mock.Mock }

func (m *mockFileAttachmentService) UploadOrReplaceFile(ctx context.Context, existing *fileattachment.FileAttachment,
	in fileattachment.UploadFileParams) (*fileattachment.FileAttachment, error) {
	args := m.Called(ctx, existing, in)
	att, _ := args.Get(0).(*fileattachment.FileAttachment)
	return att, args.Error(1)
}

type harness struct {
	repo      *mockRepository
	roomUser  *mockRoomUserService
	favourite *mockFavouriteService
	file      *mockFileAttachmentService
	svc       *Service
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	db := testutil.NewIntegrationDB(t)
	repo := new(mockRepository)
	ru := new(mockRoomUserService)
	fav := new(mockFavouriteService)
	file := new(mockFileAttachmentService)
	return &harness{
		repo:      repo,
		roomUser:  ru,
		favourite: fav,
		file:      file,
		svc:       NewService(repo, ru, fav, db, NewAuthz(ru), file),
	}
}

func TestService_FindByID(t *testing.T) {
	t.Run("success: returns room with tags from repo", func(t *testing.T) {
		h := newHarness(t)
		want := &Room{Name: "Main Stage"}
		tags := map[uint][]tag.Tag{1: {{Name: "jazz"}}}
		h.repo.On("FindByID", uint(1)).Return(want, nil)
		h.roomUser.On("FindBy", uint(7), uint(1)).Return(&roomuser.RoomUser{}, nil)
		h.repo.On("LoadTagsForRooms", []uint{1}).Return(tags, nil)
		h.roomUser.On("CountByRoomIDs", []uint{0}, mock.Anything).Return(map[uint]int64{}, nil).Times(2)

		got, err := h.svc.FindByID(1, 7)

		require.NoError(t, err)
		assert.Same(t, want, got)
		assert.Equal(t, tags[1], got.Tags)
		h.repo.AssertExpectations(t)
		h.roomUser.AssertExpectations(t)
	})

	t.Run("failure: non-member gets ErrForbidden", func(t *testing.T) {
		h := newHarness(t)
		h.repo.On("FindByID", uint(1)).Return(&Room{Name: "Main Stage"}, nil)
		h.roomUser.On("FindBy", uint(7), uint(1)).Return(nil, nil)

		got, err := h.svc.FindByID(1, 7)

		require.Nil(t, got)
		require.ErrorIs(t, err, httpx.ErrForbidden)
		h.repo.AssertExpectations(t)
		h.roomUser.AssertExpectations(t)
	})

	t.Run("failure: FindBy error is propagated", func(t *testing.T) {
		h := newHarness(t)
		findErr := errors.New("db down")
		h.repo.On("FindByID", uint(1)).Return(&Room{Name: "Main Stage"}, nil)
		h.roomUser.On("FindBy", uint(7), uint(1)).Return(nil, findErr)

		got, err := h.svc.FindByID(1, 7)

		require.Nil(t, got)
		require.ErrorIs(t, err, findErr)
		h.repo.AssertExpectations(t)
		h.roomUser.AssertExpectations(t)
	})

	t.Run("failure: LoadTagsForRooms error is propagated", func(t *testing.T) {
		h := newHarness(t)
		tagsErr := errors.New("tags failed")
		h.repo.On("FindByID", uint(99)).Return(&Room{Name: "Main Stage"}, nil)
		h.roomUser.On("FindBy", uint(7), uint(99)).Return(&roomuser.RoomUser{}, nil)
		h.repo.On("LoadTagsForRooms", []uint{99}).Return(nil, tagsErr)

		got, err := h.svc.FindByID(99, 7)

		require.Nil(t, got)
		require.ErrorIs(t, err, tagsErr)
		h.repo.AssertExpectations(t)
		h.roomUser.AssertExpectations(t)
	})

	t.Run("failure: repo error propagated", func(t *testing.T) {
		h := newHarness(t)
		repoErr := errors.New("not found")
		h.repo.On("FindByID", uint(99)).Return(nil, repoErr)

		got, err := h.svc.FindByID(99, 7)

		require.Nil(t, got)
		require.ErrorIs(t, err, repoErr)
		h.repo.AssertExpectations(t)
	})
}

func TestService_Create(t *testing.T) {
	t.Run("success: creates room and adds creator as owner in same tx", func(t *testing.T) {
		h := newHarness(t)
		input := &CreateRoomParams{Name: "New Room", CreatorID: 5}
		created := &Room{Name: "New Room"}
		created.ID = 10

		h.repo.On("Create", mock.AnythingOfType("*gorm.DB"), input).Return(created, nil)
		h.roomUser.On("Create", mock.AnythingOfType("*gorm.DB"), uint(5), uint(10), role.RoleOwner).
			Return(&roomuser.RoomUser{}, nil)
		h.roomUser.On("CountByRoomIDs", []uint{10}, mock.Anything).Return(map[uint]int64{}, nil).Times(2)

		got, err := h.svc.Create(input)

		require.NoError(t, err)
		assert.Same(t, created, got)
		h.repo.AssertExpectations(t)
		h.roomUser.AssertExpectations(t)
	})

	t.Run("failure: Create error rolls back and room creation is not returned", func(t *testing.T) {
		h := newHarness(t)
		input := &CreateRoomParams{Name: "New Room", CreatorID: 5}
		created := &Room{Name: "New Room"}
		created.ID = 10

		h.repo.On("Create", mock.AnythingOfType("*gorm.DB"), input).Return(created, nil)
		addErr := errors.New("failed to add owner")
		h.roomUser.On("Create", mock.AnythingOfType("*gorm.DB"), uint(5), uint(10), role.RoleOwner).
			Return(nil, addErr)

		got, err := h.svc.Create(input)

		require.Nil(t, got)
		require.ErrorIs(t, err, addErr)
		h.repo.AssertExpectations(t)
		h.roomUser.AssertExpectations(t)
	})
}

func TestService_Update(t *testing.T) {
	t.Run("success: returns updated room from repo", func(t *testing.T) {
		h := newHarness(t)
		input := &UpdateRoomParams{Name: "Renamed"}
		updated := &Room{Name: "Renamed"}
		h.roomUser.On("HasRoles", uint(1), uint(3), []role.RoleName{role.RoleOwner, role.RoleAdmin}).Return(true, nil)
		h.repo.On("FindByID", uint(3)).Return(&Room{PrivateCode: nil}, nil)
		h.repo.On("Update", uint(3), input).Return(updated, nil)
		h.roomUser.On("CountByRoomIDs", []uint{0}, mock.Anything).Return(map[uint]int64{}, nil).Times(2)

		got, err := h.svc.Update(3, 1, input)

		require.NoError(t, err)
		assert.Same(t, updated, got)
		h.repo.AssertExpectations(t)
		h.roomUser.AssertExpectations(t)
	})

	t.Run("failure: actor without permission returns ErrForbidden", func(t *testing.T) {
		h := newHarness(t)
		input := &UpdateRoomParams{Name: "Renamed"}
		h.roomUser.On("HasRoles", uint(1), uint(3), []role.RoleName{role.RoleOwner, role.RoleAdmin}).Return(false, nil)

		got, err := h.svc.Update(3, 1, input)

		require.Nil(t, got)
		require.ErrorIs(t, err, httpx.ErrForbidden)
		h.roomUser.AssertExpectations(t)
	})

	t.Run("failure: repo error propagated", func(t *testing.T) {
		h := newHarness(t)
		input := &UpdateRoomParams{Name: "Renamed"}
		repoErr := errors.New("room not found")
		h.roomUser.On("HasRoles", uint(1), uint(3), []role.RoleName{role.RoleOwner, role.RoleAdmin}).Return(true, nil)
		h.repo.On("FindByID", uint(3)).Return(nil, repoErr)

		got, err := h.svc.Update(3, 1, input)

		require.Nil(t, got)
		require.ErrorIs(t, err, repoErr)
		h.repo.AssertExpectations(t)
		h.roomUser.AssertExpectations(t)
	})
}

func TestService_List(t *testing.T) {
	t.Run("success: returns rooms with loaded tags and total count", func(t *testing.T) {
		h := newHarness(t)
		filter := RoomFilter{}
		sort := listopts.Sort{}
		p := listopts.Pagination{Page: 1, PageSize: 10}
		rooms := []Room{{Name: "A"}, {Name: "B"}}
		rooms[0].ID = 1
		rooms[1].ID = 2
		rooms[0].Tags = []tag.Tag{{Name: "jazz"}}
		tags := map[uint][]tag.Tag{1: rooms[0].Tags}

		h.repo.On("List", filter, sort, p).Return(rooms, nil)
		h.repo.On("LoadTagsForRooms", []uint{1, 2}).Return(tags, nil)
		h.repo.On("Count", filter).Return(int64(2), nil)
		h.roomUser.On("CountByRoomIDs", []uint{1, 2}, mock.Anything).Return(map[uint]int64{}, nil).Times(2)

		got, count, err := h.svc.List(filter, sort, p)

		require.NoError(t, err)
		assert.Equal(t, rooms, got)
		assert.Equal(t, int64(2), count)
		h.repo.AssertExpectations(t)
	})

	t.Run("failure: List error short-circuits before LoadTagsForRooms and Count are called", func(t *testing.T) {
		h := newHarness(t)
		filter := RoomFilter{}
		sort := listopts.Sort{}
		p := listopts.Pagination{Page: 1, PageSize: 10}
		listErr := errors.New("query failed")

		h.repo.On("List", filter, sort, p).Return(nil, listErr)

		got, count, err := h.svc.List(filter, sort, p)

		require.Nil(t, got)
		require.Zero(t, count)
		require.ErrorIs(t, err, listErr)
		h.repo.AssertNotCalled(t, "LoadTagsForRooms", mock.Anything)
		h.repo.AssertNotCalled(t, "Count", mock.Anything)
		h.repo.AssertExpectations(t)
	})

	t.Run("failure: LoadTagsForRooms error short-circuits before Count is called", func(t *testing.T) {
		h := newHarness(t)
		filter := RoomFilter{}
		sort := listopts.Sort{}
		p := listopts.Pagination{Page: 1, PageSize: 10}
		rooms := []Room{{Name: "A"}}
		rooms[0].ID = 1
		loadErr := errors.New("tags failed")

		h.repo.On("List", filter, sort, p).Return(rooms, nil)
		h.repo.On("LoadTagsForRooms", []uint{1}).Return(nil, loadErr)

		got, count, err := h.svc.List(filter, sort, p)

		require.Nil(t, got)
		require.Zero(t, count)
		require.ErrorIs(t, err, loadErr)
		h.repo.AssertNotCalled(t, "Count", mock.Anything)
		h.repo.AssertExpectations(t)
	})

	t.Run("failure: Count error after successful LoadTagsForRooms still fails the call", func(t *testing.T) {
		h := newHarness(t)
		filter := RoomFilter{}
		sort := listopts.Sort{}
		p := listopts.Pagination{Page: 1, PageSize: 10}
		rooms := []Room{{Name: "A"}}
		rooms[0].ID = 1
		countErr := errors.New("count query failed")

		h.repo.On("List", filter, sort, p).Return(rooms, nil)
		h.repo.On("LoadTagsForRooms", []uint{1}).Return(map[uint][]tag.Tag{}, nil)
		h.repo.On("Count", filter).Return(int64(0), countErr)

		got, count, err := h.svc.List(filter, sort, p)

		require.Nil(t, got)
		require.Zero(t, count)
		require.ErrorIs(t, err, countErr)
		h.repo.AssertExpectations(t)
	})
}

func TestService_ViewerContext(t *testing.T) {
	t.Run("success: returns membership and favourite status", func(t *testing.T) {
		h := newHarness(t)
		ru := &roomuser.RoomUser{UserID: 7, RoomID: 4}
		h.roomUser.On("MapByUserAndRoomIDs", uint(7), []uint{4}).Return(map[uint]*roomuser.RoomUser{4: ru}, nil)
		h.favourite.On("FavouritedRoomIDs", uint(7), []uint{4}).Return(map[uint]bool{4: true}, nil)

		got, err := h.svc.ViewerContext(4, 7)

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Same(t, ru, got.RoomUser)
		assert.True(t, got.IsFavourited)
		h.roomUser.AssertExpectations(t)
		h.favourite.AssertExpectations(t)
	})

	t.Run("success: non-member non-favourited viewer", func(t *testing.T) {
		h := newHarness(t)
		h.roomUser.On("MapByUserAndRoomIDs", uint(7), []uint{4}).Return(map[uint]*roomuser.RoomUser{}, nil)
		h.favourite.On("FavouritedRoomIDs", uint(7), []uint{4}).Return(map[uint]bool{}, nil)

		got, err := h.svc.ViewerContext(4, 7)

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Nil(t, got.RoomUser)
		assert.False(t, got.IsFavourited)
	})

	t.Run("failure: roomUserService error propagated", func(t *testing.T) {
		h := newHarness(t)
		findErr := errors.New("membership lookup failed")
		h.roomUser.On("MapByUserAndRoomIDs", uint(7), []uint{4}).Return(nil, findErr)

		got, err := h.svc.ViewerContext(4, 7)

		require.Nil(t, got)
		require.ErrorIs(t, err, findErr)
		h.favourite.AssertNotCalled(t, "FavouritedRoomIDs", mock.Anything, mock.Anything)
	})

	t.Run("failure: favouriteService error propagated", func(t *testing.T) {
		h := newHarness(t)
		favErr := errors.New("favourite lookup failed")
		h.roomUser.On("MapByUserAndRoomIDs", uint(7), []uint{4}).Return(map[uint]*roomuser.RoomUser{}, nil)
		h.favourite.On("FavouritedRoomIDs", uint(7), []uint{4}).Return(nil, favErr)

		got, err := h.svc.ViewerContext(4, 7)

		require.Nil(t, got)
		require.ErrorIs(t, err, favErr)
	})
}

func TestService_ViewerContexts(t *testing.T) {
	t.Run("success: returns per-room viewer context", func(t *testing.T) {
		h := newHarness(t)
		ru := &roomuser.RoomUser{UserID: 7, RoomID: 4}
		h.roomUser.On("MapByUserAndRoomIDs", uint(7), []uint{4, 5}).
			Return(map[uint]*roomuser.RoomUser{4: ru}, nil)
		h.favourite.On("FavouritedRoomIDs", uint(7), []uint{4, 5}).Return(map[uint]bool{5: true}, nil)

		got, err := h.svc.ViewerContexts([]uint{4, 5}, 7)

		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Same(t, ru, got[4].RoomUser)
		assert.False(t, got[4].IsFavourited)
		assert.Nil(t, got[5].RoomUser)
		assert.True(t, got[5].IsFavourited)
	})
}

func TestService_UpdatePrivateCode(t *testing.T) {
	t.Run("success: admin updates the private code", func(t *testing.T) {
		h := newHarness(t)
		h.roomUser.On("HasRoles", uint(1), uint(5), []role.RoleName{role.RoleOwner, role.RoleAdmin}).Return(true, nil)
		h.repo.On("UpdatePrivateCode", uint(5), mock.AnythingOfType("string")).Return(nil)

		err := h.svc.UpdatePrivateCode(5, 1)

		require.NoError(t, err)
		h.roomUser.AssertExpectations(t)
		h.repo.AssertExpectations(t)
	})

	t.Run("failure: non-admin gets ErrForbidden", func(t *testing.T) {
		h := newHarness(t)
		h.roomUser.On("HasRoles", uint(1), uint(5), []role.RoleName{role.RoleOwner, role.RoleAdmin}).Return(false, nil)

		err := h.svc.UpdatePrivateCode(5, 1)

		require.ErrorIs(t, err, httpx.ErrForbidden)
		h.repo.AssertNotCalled(t, "UpdatePrivateCode", mock.Anything, mock.Anything)
		h.roomUser.AssertExpectations(t)
	})

	t.Run("failure: HasRoles error is propagated", func(t *testing.T) {
		h := newHarness(t)
		permErr := errors.New("permission check failed")
		h.roomUser.On("HasRoles", uint(1), uint(5), []role.RoleName{role.RoleOwner, role.RoleAdmin}).Return(false, permErr)

		err := h.svc.UpdatePrivateCode(5, 1)

		require.ErrorIs(t, err, permErr)
		h.repo.AssertNotCalled(t, "UpdatePrivateCode", mock.Anything, mock.Anything)
		h.roomUser.AssertExpectations(t)
	})

	t.Run("failure: repo error is propagated", func(t *testing.T) {
		h := newHarness(t)
		h.roomUser.On("HasRoles", uint(1), uint(5), []role.RoleName{role.RoleOwner, role.RoleAdmin}).Return(true, nil)
		repoErr := errors.New("not found")
		h.repo.On("UpdatePrivateCode", uint(5), mock.AnythingOfType("string")).Return(repoErr)

		err := h.svc.UpdatePrivateCode(5, 1)

		require.ErrorIs(t, err, repoErr)
		h.roomUser.AssertExpectations(t)
		h.repo.AssertExpectations(t)
	})
}

func TestService_AddRoomUser(t *testing.T) {
	t.Run("success: adds user to a public room", func(t *testing.T) {
		h := newHarness(t)
		created := &roomuser.RoomUser{UserID: 1, RoomID: 4}
		h.roomUser.On("IsBlocked", uint(4), uint(1)).Return(false, nil)
		h.roomUser.On("FindBy", uint(1), uint(4)).Return(nil, nil)
		h.repo.On("FindByID", uint(4)).Return(&Room{Type: RoomTypePublic}, nil)
		h.roomUser.On("Create", (*gorm.DB)(nil), uint(1), uint(4), role.RoleListener).Return(created, nil)

		got, err := h.svc.AddRoomUser(4, 1, "")

		require.NoError(t, err)
		assert.Same(t, created, got)
		h.repo.AssertExpectations(t)
		h.roomUser.AssertExpectations(t)
	})

	t.Run("success: rejoins existing member", func(t *testing.T) {
		h := newHarness(t)
		existing := &roomuser.RoomUser{UserID: 1, RoomID: 4}
		h.roomUser.On("IsBlocked", uint(4), uint(1)).Return(false, nil)
		h.roomUser.On("FindBy", uint(1), uint(4)).Return(existing, nil)
		h.roomUser.On("Rejoin", existing).Return(nil)

		got, err := h.svc.AddRoomUser(4, 1, "")

		require.NoError(t, err)
		assert.Same(t, existing, got)
		h.roomUser.AssertExpectations(t)
	})

	t.Run("failure: blocked user returns ErrUserBlocked", func(t *testing.T) {
		h := newHarness(t)
		h.roomUser.On("IsBlocked", uint(4), uint(1)).Return(true, nil)

		got, err := h.svc.AddRoomUser(4, 1, "")

		require.Nil(t, got)
		require.ErrorIs(t, err, httpx.ErrUserBlocked)
		h.roomUser.AssertNotCalled(t, "FindBy", mock.Anything, mock.Anything)
		h.repo.AssertNotCalled(t, "FindByID", mock.Anything)
		h.roomUser.AssertExpectations(t)
	})

	t.Run("failure: wrong private code returns ErrForbidden", func(t *testing.T) {
		h := newHarness(t)
		code := "secret-123"
		h.roomUser.On("IsBlocked", uint(4), uint(1)).Return(false, nil)
		h.roomUser.On("FindBy", uint(1), uint(4)).Return(nil, nil)
		h.repo.On("FindByID", uint(4)).Return(&Room{Type: RoomTypePrivate, PrivateCode: &code}, nil)

		got, err := h.svc.AddRoomUser(4, 1, "wrong")

		require.Nil(t, got)
		require.ErrorIs(t, err, httpx.ErrForbidden)
		h.repo.AssertExpectations(t)
		h.roomUser.AssertExpectations(t)
	})

	t.Run("failure: room lookup error is propagated", func(t *testing.T) {
		h := newHarness(t)
		findErr := errors.New("db down")
		h.roomUser.On("IsBlocked", uint(4), uint(1)).Return(false, nil)
		h.roomUser.On("FindBy", uint(1), uint(4)).Return(nil, nil)
		h.repo.On("FindByID", uint(4)).Return(nil, findErr)

		got, err := h.svc.AddRoomUser(4, 1, "")

		require.Nil(t, got)
		require.ErrorIs(t, err, findErr)
		h.repo.AssertExpectations(t)
		h.roomUser.AssertExpectations(t)
	})
}
