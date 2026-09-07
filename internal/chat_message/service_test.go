package chatmessage

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"sound-stage-backend/internal/pkg/httpx"
	"sound-stage-backend/internal/pkg/listopts"
	"sound-stage-backend/internal/role"
	roomuser "sound-stage-backend/internal/room_user"
)

type mockRepository struct{ mock.Mock }

func (m *mockRepository) Create(input *CreateChatMessageParams) (*ChatMessage, error) {
	args := m.Called(input)
	msg, _ := args.Get(0).(*ChatMessage)
	return msg, args.Error(1)
}

func (m *mockRepository) List(filter ChatMessageFilter, p listopts.Pagination) ([]ChatMessage, error) {
	args := m.Called(filter, p)
	msgs, _ := args.Get(0).([]ChatMessage)
	return msgs, args.Error(1)
}

func (m *mockRepository) Count(filter ChatMessageFilter) (int64, error) {
	args := m.Called(filter)
	return args.Get(0).(int64), args.Error(1)
}

type mockRoomUserService struct{ mock.Mock }

func (m *mockRoomUserService) FindBy(userID, roomID uint) (*roomuser.RoomUser, error) {
	args := m.Called(userID, roomID)
	ru, _ := args.Get(0).(*roomuser.RoomUser)
	return ru, args.Error(1)
}

type harness struct {
	repo      *mockRepository
	roomUsers *mockRoomUserService
	svc       *Service
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	repo := new(mockRepository)
	ru := new(mockRoomUserService)
	return &harness{
		repo:      repo,
		roomUsers: ru,
		svc:       NewService(repo, NewAuthz(ru)),
	}
}

func TestService_Create(t *testing.T) {
	t.Run("success: member creates a message", func(t *testing.T) {
		h := newHarness(t)
		input := &CreateChatMessageParams{RoomID: 1, UserID: 2, Content: "hello"}
		created := &ChatMessage{RoomID: 1, UserID: 2, Content: "hello"}

		h.roomUsers.On("FindBy", uint(2), uint(1)).Return(&roomuser.RoomUser{Role: role.Role{Name: role.RoleListener}}, nil)
		h.repo.On("Create", input).Return(created, nil)

		got, err := h.svc.Create(input)

		require.NoError(t, err)
		assert.Same(t, created, got)
		h.roomUsers.AssertExpectations(t)
		h.repo.AssertExpectations(t)
	})

	t.Run("success: admin creates a pinned message", func(t *testing.T) {
		h := newHarness(t)
		input := &CreateChatMessageParams{RoomID: 1, UserID: 2, Content: "pinned", IsPinned: true}
		created := &ChatMessage{RoomID: 1, UserID: 2, Content: "pinned", IsPinned: true}

		h.roomUsers.On("FindBy", uint(2), uint(1)).Return(&roomuser.RoomUser{Role: role.Role{Name: role.RoleAdmin}}, nil)
		h.repo.On("Create", input).Return(created, nil)

		got, err := h.svc.Create(input)

		require.NoError(t, err)
		assert.Same(t, created, got)
		h.roomUsers.AssertExpectations(t)
		h.repo.AssertExpectations(t)
	})

	t.Run("failure: non-member is forbidden", func(t *testing.T) {
		h := newHarness(t)
		input := &CreateChatMessageParams{RoomID: 1, UserID: 2, Content: "hello"}

		h.roomUsers.On("FindBy", uint(2), uint(1)).Return(nil, nil)

		got, err := h.svc.Create(input)

		require.Nil(t, got)
		require.ErrorIs(t, err, httpx.ErrForbidden)
		h.roomUsers.AssertExpectations(t)
		h.repo.AssertNotCalled(t, "Create", mock.Anything)
	})

	t.Run("failure: non-admin cannot pin", func(t *testing.T) {
		h := newHarness(t)
		input := &CreateChatMessageParams{RoomID: 1, UserID: 2, Content: "pinned", IsPinned: true}

		h.roomUsers.On("FindBy", uint(2), uint(1)).Return(&roomuser.RoomUser{Role: role.Role{Name: role.RoleListener}}, nil)

		got, err := h.svc.Create(input)

		require.Nil(t, got)
		require.ErrorIs(t, err, httpx.ErrForbidden)
		h.roomUsers.AssertExpectations(t)
		h.repo.AssertNotCalled(t, "Create", mock.Anything)
	})

	t.Run("failure: repo error is propagated", func(t *testing.T) {
		h := newHarness(t)
		input := &CreateChatMessageParams{RoomID: 1, UserID: 2, Content: "hello"}
		repoErr := errors.New("db down")

		h.roomUsers.On("FindBy", uint(2), uint(1)).Return(&roomuser.RoomUser{Role: role.Role{Name: role.RoleListener}}, nil)
		h.repo.On("Create", input).Return(nil, repoErr)

		got, err := h.svc.Create(input)

		require.Nil(t, got)
		require.ErrorIs(t, err, repoErr)
		h.repo.AssertExpectations(t)
		h.roomUsers.AssertExpectations(t)
	})
}

func TestService_List(t *testing.T) {
	t.Run("success: returns messages and count", func(t *testing.T) {
		h := newHarness(t)
		filter := ChatMessageFilter{RoomID: 1}
		p := listopts.Pagination{Page: 1, PageSize: 10}
		messages := []ChatMessage{{Content: "hi"}}

		h.roomUsers.On("FindBy", uint(2), uint(1)).Return(&roomuser.RoomUser{Role: role.Role{Name: role.RoleListener}}, nil)
		h.repo.On("List", filter, p).Return(messages, nil)
		h.repo.On("Count", filter).Return(int64(1), nil)

		got, count, err := h.svc.List(2, filter, p)

		require.NoError(t, err)
		assert.Equal(t, messages, got)
		assert.Equal(t, int64(1), count)
		h.roomUsers.AssertExpectations(t)
		h.repo.AssertExpectations(t)
	})

	t.Run("failure: non-member is forbidden", func(t *testing.T) {
		h := newHarness(t)
		filter := ChatMessageFilter{RoomID: 1}
		p := listopts.Pagination{Page: 1, PageSize: 10}

		h.roomUsers.On("FindBy", uint(2), uint(1)).Return(nil, nil)

		got, count, err := h.svc.List(2, filter, p)

		require.Nil(t, got)
		require.Zero(t, count)
		require.ErrorIs(t, err, httpx.ErrForbidden)
		h.roomUsers.AssertExpectations(t)
		h.repo.AssertNotCalled(t, "List", mock.Anything, mock.Anything)
	})

	t.Run("failure: repo List error is propagated", func(t *testing.T) {
		h := newHarness(t)
		filter := ChatMessageFilter{RoomID: 1}
		p := listopts.Pagination{Page: 1, PageSize: 10}
		listErr := errors.New("list failed")

		h.roomUsers.On("FindBy", uint(2), uint(1)).Return(&roomuser.RoomUser{Role: role.Role{Name: role.RoleListener}}, nil)
		h.repo.On("List", filter, p).Return(nil, listErr)

		got, count, err := h.svc.List(2, filter, p)

		require.Nil(t, got)
		require.Zero(t, count)
		require.ErrorIs(t, err, listErr)
		h.repo.AssertNotCalled(t, "Count", mock.Anything)
	})

	t.Run("failure: repo Count error is propagated", func(t *testing.T) {
		h := newHarness(t)
		filter := ChatMessageFilter{RoomID: 1}
		p := listopts.Pagination{Page: 1, PageSize: 10}
		countErr := errors.New("count failed")
		messages := []ChatMessage{{Content: "hi"}}

		h.roomUsers.On("FindBy", uint(2), uint(1)).Return(&roomuser.RoomUser{Role: role.Role{Name: role.RoleListener}}, nil)
		h.repo.On("List", filter, p).Return(messages, nil)
		h.repo.On("Count", filter).Return(int64(0), countErr)

		got, count, err := h.svc.List(2, filter, p)

		require.Nil(t, got)
		require.Zero(t, count)
		require.ErrorIs(t, err, countErr)
		h.repo.AssertExpectations(t)
	})
}
