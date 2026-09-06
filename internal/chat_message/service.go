package chatmessage

import (
	"sound-stage-backend/internal/pkg/httpx"
	"sound-stage-backend/internal/pkg/listopts"
	roomuser "sound-stage-backend/internal/room_user"
)

type repository interface {
	Create(input *CreateChatMessageParams) (*ChatMessage, error)
	List(filter ChatMessageFilter, p listopts.Pagination) ([]ChatMessage, error)
	Count(filter ChatMessageFilter) (int64, error)
}

type roomUserService interface {
	FindBy(userID uint, roomID uint) (*roomuser.RoomUser, error)
}

type Service struct {
	repo      repository
	roomUsers roomUserService
}

func NewService(r repository, roomUsers roomUserService) *Service {
	return &Service{repo: r, roomUsers: roomUsers}
}

func (s *Service) Create(input *CreateChatMessageParams) (*ChatMessage, error) {
	ru, err := s.roomUsers.FindBy(input.UserID, input.RoomID)
	if err != nil {
		return nil, err
	}
	if ru == nil {
		return nil, httpx.ErrForbidden
	}

	if input.IsPinned && !ru.IsAdmin() {
		return nil, httpx.ErrForbidden
	}

	msg, err := s.repo.Create(input)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func (s *Service) List(userID uint, filter ChatMessageFilter, p listopts.Pagination) ([]ChatMessage, int64, error) {
	ru, err := s.roomUsers.FindBy(userID, filter.RoomID)
	if err != nil {
		return nil, 0, err
	}
	if ru == nil {
		return nil, 0, httpx.ErrForbidden
	}

	messages, err := s.repo.List(filter, p)
	if err != nil {
		return nil, 0, err
	}

	count, err := s.repo.Count(filter)
	if err != nil {
		return nil, 0, err
	}

	return messages, count, nil
}
