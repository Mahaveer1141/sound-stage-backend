package chatmessage

import (
	"sound-stage-backend/internal/pkg/listopts"
)

type repository interface {
	Create(input *CreateChatMessageParams) (*ChatMessage, error)
	List(filter ChatMessageFilter, p listopts.Pagination) ([]ChatMessage, error)
	Count(filter ChatMessageFilter) (int64, error)
}

type authorizer interface {
	CanCreate(input *CreateChatMessageParams) error
	CanList(userID, roomID uint) error
}

type Service struct {
	repo  repository
	authz authorizer
}

func NewService(r repository, authz authorizer) *Service {
	return &Service{repo: r, authz: authz}
}

func (s *Service) Create(input *CreateChatMessageParams) (*ChatMessage, error) {
	if err := s.authz.CanCreate(input); err != nil {
		return nil, err
	}

	msg, err := s.repo.Create(input)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func (s *Service) List(userID uint, filter ChatMessageFilter, p listopts.Pagination) ([]ChatMessage, int64, error) {
	if err := s.authz.CanList(userID, filter.RoomID); err != nil {
		return nil, 0, err
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
