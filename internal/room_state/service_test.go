package roomstate

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"sound-stage-backend/internal/pkg/listopts"
	"sound-stage-backend/internal/ws"
)

type mockRepository struct{ mock.Mock }

func (m *mockRepository) SetMuted(ctx context.Context, roomID, userID uint, isMuted bool) (*ParticipantState, error) {
	args := m.Called(ctx, roomID, userID, isMuted)
	s, _ := args.Get(0).(*ParticipantState)
	return s, args.Error(1)
}
func (m *mockRepository) SetHandRaised(ctx context.Context, roomID, userID uint, isHandRaised bool) (*ParticipantState, error) {
	args := m.Called(ctx, roomID, userID, isHandRaised)
	s, _ := args.Get(0).(*ParticipantState)
	return s, args.Error(1)
}
func (m *mockRepository) GetParticipantStates(ctx context.Context, roomID uint, userIDs []uint) (map[uint]*ParticipantState, error) {
	args := m.Called(ctx, roomID, userIDs)
	res, _ := args.Get(0).(map[uint]*ParticipantState)
	return res, args.Error(1)
}
func (m *mockRepository) DeleteParticipantState(ctx context.Context, roomID, userID uint) error {
	args := m.Called(ctx, roomID, userID)
	return args.Error(0)
}
func (m *mockRepository) GetRaisedHands(ctx context.Context, roomID uint, p listopts.Pagination) ([]uint, error) {
	args := m.Called(ctx, roomID, p)
	ids, _ := args.Get(0).([]uint)
	return ids, args.Error(1)
}
func (m *mockRepository) CountRaisedHands(ctx context.Context, roomID uint) (int64, error) {
	args := m.Called(ctx, roomID)
	return args.Get(0).(int64), args.Error(1)
}

type mockPublisher struct{ mock.Mock }

func (m *mockPublisher) Publish(ctx context.Context, roomID, userID uint, event ws.EventName, payload any) error {
	args := m.Called(ctx, roomID, userID, event, payload)
	return args.Error(0)
}

type serviceHarness struct {
	repo *mockRepository
	pub  *mockPublisher
	svc  *Service
}

func newServiceHarness(t *testing.T) *serviceHarness {
	t.Helper()
	repo := new(mockRepository)
	pub := new(mockPublisher)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return &serviceHarness{
		repo: repo,
		pub:  pub,
		svc:  NewService(repo, pub, logger),
	}
}

func TestService_Leave(t *testing.T) {
	t.Run("success: deletes participant state", func(t *testing.T) {
		h := newServiceHarness(t)
		h.repo.On("DeleteParticipantState", mock.Anything, uint(4), uint(42)).Return(nil)

		err := h.svc.Leave(context.Background(), 4, 42)

		require.NoError(t, err)
		h.repo.AssertExpectations(t)
	})

	t.Run("failure: repo error is propagated", func(t *testing.T) {
		h := newServiceHarness(t)
		repoErr := errors.New("redis down")
		h.repo.On("DeleteParticipantState", mock.Anything, uint(4), uint(42)).Return(repoErr)

		err := h.svc.Leave(context.Background(), 4, 42)

		require.ErrorIs(t, err, repoErr)
		h.repo.AssertExpectations(t)
	})
}

func TestService_SetMuted(t *testing.T) {
	t.Run("success: updates state and publishes set_muted event", func(t *testing.T) {
		h := newServiceHarness(t)
		state := &ParticipantState{UserID: 42, IsMuted: true}
		h.repo.On("SetMuted", mock.Anything, uint(4), uint(42), true).Return(state, nil)
		h.pub.On("Publish", mock.Anything, uint(4), uint(42), ws.EventSetMuted, state).Return(nil)

		err := h.svc.SetMuted(context.Background(), 4, 42, true)

		require.NoError(t, err)
		h.repo.AssertExpectations(t)
		h.pub.AssertExpectations(t)
	})

	t.Run("failure: repo error short-circuits, no publish", func(t *testing.T) {
		h := newServiceHarness(t)
		repoErr := errors.New("redis down")
		h.repo.On("SetMuted", mock.Anything, uint(4), uint(42), true).Return(nil, repoErr)

		err := h.svc.SetMuted(context.Background(), 4, 42, true)

		require.ErrorIs(t, err, repoErr)
		h.pub.AssertNotCalled(t, "Publish", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		h.repo.AssertExpectations(t)
	})

	t.Run("failure: publish error is propagated", func(t *testing.T) {
		h := newServiceHarness(t)
		state := &ParticipantState{UserID: 42, IsMuted: true}
		pubErr := errors.New("publish failed")
		h.repo.On("SetMuted", mock.Anything, uint(4), uint(42), true).Return(state, nil)
		h.pub.On("Publish", mock.Anything, uint(4), uint(42), ws.EventSetMuted, state).Return(pubErr)

		err := h.svc.SetMuted(context.Background(), 4, 42, true)

		require.ErrorIs(t, err, pubErr)
		h.repo.AssertExpectations(t)
		h.pub.AssertExpectations(t)
	})
}

func TestService_SetHandRaised(t *testing.T) {
	t.Run("success: updates state and publishes set_hand_raised event", func(t *testing.T) {
		h := newServiceHarness(t)
		state := &ParticipantState{UserID: 42, IsHandRaised: true}
		h.repo.On("SetHandRaised", mock.Anything, uint(4), uint(42), true).Return(state, nil)
		h.pub.On("Publish", mock.Anything, uint(4), uint(42), ws.EventSetHandRaised, state).Return(nil)

		err := h.svc.SetHandRaised(context.Background(), 4, 42, true)

		require.NoError(t, err)
		h.repo.AssertExpectations(t)
		h.pub.AssertExpectations(t)
	})

	t.Run("failure: repo error short-circuits, no publish", func(t *testing.T) {
		h := newServiceHarness(t)
		repoErr := errors.New("redis down")
		h.repo.On("SetHandRaised", mock.Anything, uint(4), uint(42), true).Return(nil, repoErr)

		err := h.svc.SetHandRaised(context.Background(), 4, 42, true)

		require.ErrorIs(t, err, repoErr)
		h.pub.AssertNotCalled(t, "Publish", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		h.repo.AssertExpectations(t)
	})

	t.Run("failure: publish error is propagated", func(t *testing.T) {
		h := newServiceHarness(t)
		state := &ParticipantState{UserID: 42, IsHandRaised: true}
		pubErr := errors.New("publish failed")
		h.repo.On("SetHandRaised", mock.Anything, uint(4), uint(42), true).Return(state, nil)
		h.pub.On("Publish", mock.Anything, uint(4), uint(42), ws.EventSetHandRaised, state).Return(pubErr)

		err := h.svc.SetHandRaised(context.Background(), 4, 42, true)

		require.ErrorIs(t, err, pubErr)
		h.repo.AssertExpectations(t)
		h.pub.AssertExpectations(t)
	})
}

func TestService_GetRaisedHands(t *testing.T) {
	t.Run("success: returns raised hand user ids", func(t *testing.T) {
		h := newServiceHarness(t)
		p := listopts.Pagination{Page: 1, PageSize: 10}
		h.repo.On("GetRaisedHands", mock.Anything, uint(4), p).Return([]uint{7, 9}, nil)

		got, err := h.svc.GetRaisedHands(context.Background(), 4, p)

		require.NoError(t, err)
		assert.Equal(t, []uint{7, 9}, got)
		h.repo.AssertExpectations(t)
	})

	t.Run("failure: repo error is propagated", func(t *testing.T) {
		h := newServiceHarness(t)
		p := listopts.Pagination{Page: 1, PageSize: 10}
		repoErr := errors.New("redis down")
		h.repo.On("GetRaisedHands", mock.Anything, uint(4), p).Return(nil, repoErr)

		got, err := h.svc.GetRaisedHands(context.Background(), 4, p)

		require.Nil(t, got)
		require.ErrorIs(t, err, repoErr)
		h.repo.AssertExpectations(t)
	})
}

func TestService_CountRaisedHands(t *testing.T) {
	t.Run("success: returns raised hands count", func(t *testing.T) {
		h := newServiceHarness(t)
		h.repo.On("CountRaisedHands", mock.Anything, uint(4)).Return(int64(5), nil)

		got, err := h.svc.CountRaisedHands(context.Background(), 4)

		require.NoError(t, err)
		assert.Equal(t, int64(5), got)
		h.repo.AssertExpectations(t)
	})

	t.Run("failure: repo error is propagated", func(t *testing.T) {
		h := newServiceHarness(t)
		repoErr := errors.New("redis down")
		h.repo.On("CountRaisedHands", mock.Anything, uint(4)).Return(int64(0), repoErr)

		got, err := h.svc.CountRaisedHands(context.Background(), 4)

		require.Zero(t, got)
		require.ErrorIs(t, err, repoErr)
		h.repo.AssertExpectations(t)
	})
}

func TestService_GetParticipantStates(t *testing.T) {
	t.Run("success: returns states map from repo", func(t *testing.T) {
		h := newServiceHarness(t)
		states := map[uint]*ParticipantState{42: {UserID: 42, IsMuted: true}}
		h.repo.On("GetParticipantStates", mock.Anything, uint(4), []uint{42}).Return(states, nil)

		got, err := h.svc.GetParticipantStates(context.Background(), 4, []uint{42})

		require.NoError(t, err)
		assert.Equal(t, states, got)
		h.repo.AssertExpectations(t)
	})

	t.Run("failure: repo error is propagated", func(t *testing.T) {
		h := newServiceHarness(t)
		repoErr := errors.New("redis down")
		h.repo.On("GetParticipantStates", mock.Anything, uint(4), []uint{42}).Return(nil, repoErr)

		got, err := h.svc.GetParticipantStates(context.Background(), 4, []uint{42})

		require.Nil(t, got)
		require.ErrorIs(t, err, repoErr)
		h.repo.AssertExpectations(t)
	})
}
