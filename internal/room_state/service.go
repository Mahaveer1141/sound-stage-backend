package roomstate

import (
	"context"
	"fmt"
	"log/slog"
	"sound-stage-backend/internal/pkg/listopts"
	"sound-stage-backend/internal/ws"
)

type Repository interface {
	SetMuted(ctx context.Context, roomID, userID uint, isMuted bool) (*ParticipantState, error)
	SetHandRaised(ctx context.Context, roomID, userID uint, isHandRaised bool) (*ParticipantState, error)
	GetParticipantStates(ctx context.Context, roomID uint, userIDs []uint) (map[uint]*ParticipantState, error)
	DeleteParticipantState(ctx context.Context, roomID, userID uint) error
	DeleteRoomState(ctx context.Context, roomID uint) error
	GetRaisedHands(ctx context.Context, roomID uint, p listopts.Pagination) ([]uint, error)
	CountRaisedHands(ctx context.Context, roomID uint) (int64, error)
}

type Service struct {
	repo   Repository
	pub    Publisher
	logger *slog.Logger
}

func NewService(repo Repository, pub Publisher, logger *slog.Logger) *Service {
	return &Service{repo: repo, pub: pub, logger: logger}
}

func (s *Service) Leave(ctx context.Context, roomID, userID uint) error {
	if err := s.repo.DeleteParticipantState(ctx, roomID, userID); err != nil {
		return fmt.Errorf("roomstate: leave: %w", err)
	}
	return nil
}

func (s *Service) SetMuted(ctx context.Context, roomID, userID uint, isMuted bool) error {
	state, err := s.repo.SetMuted(ctx, roomID, userID, isMuted)
	if err != nil {
		return fmt.Errorf("roomstate: set muted: %w", err)
	}

	if err := s.pub.Publish(ctx, roomID, userID, ws.EventSetMuted, state); err != nil {
		return fmt.Errorf("roomstate: set muted: %w", err)
	}
	return nil
}

func (s *Service) SetHandRaised(ctx context.Context, roomID, userID uint, isHandRaised bool, roomUser any) error {
	state, err := s.repo.SetHandRaised(ctx, roomID, userID, isHandRaised)
	if err != nil {
		return fmt.Errorf("roomstate: set hand raised: %w", err)
	}

	payload := HandRaisedEventPayload{RoomUser: roomUser}
	if state != nil {
		payload.ParticipantState = *state
	}
	if err := s.pub.Publish(ctx, roomID, userID, ws.EventSetHandRaised, payload); err != nil {
		return fmt.Errorf("roomstate: set hand raised: %w", err)
	}
	return nil
}

func (s *Service) DeleteRoomState(ctx context.Context, roomID uint) error {
	if err := s.repo.DeleteRoomState(ctx, roomID); err != nil {
		return fmt.Errorf("roomstate: delete room state: %w", err)
	}
	return nil
}

func (s *Service) GetParticipantStates(ctx context.Context, roomID uint, userIDs []uint) (map[uint]*ParticipantState, error) {
	return s.repo.GetParticipantStates(ctx, roomID, userIDs)
}

func (s *Service) GetRaisedHands(ctx context.Context, roomID uint, p listopts.Pagination) ([]uint, error) {
	return s.repo.GetRaisedHands(ctx, roomID, p)
}

func (s *Service) CountRaisedHands(ctx context.Context, roomID uint) (int64, error) {
	return s.repo.CountRaisedHands(ctx, roomID)
}
