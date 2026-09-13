package chatmessage

import (
	"errors"
	"sound-stage-backend/internal/pkg/httpx"
	"sound-stage-backend/internal/pkg/listopts"

	"gorm.io/gorm"
)

type repository interface {
	Create(input *CreateChatMessageParams) (*ChatMessage, error)
	FindByID(id uint) (*ChatMessage, error)
	List(filter ChatMessageFilter, p listopts.Pagination) ([]ChatMessage, error)
	Count(filter ChatMessageFilter) (int64, error)
	SetPinned(id uint, pinned bool) error
}

type authorizer interface {
	CanCreate(input *CreateChatMessageParams) error
	CanList(userID, roomID uint) error
	CanUpdatePin(userID, roomID uint) error
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

func (s *Service) SetPinned(userID, roomID, messageID uint, pinned bool) (*ChatMessage, error) {
	if err := s.authz.CanUpdatePin(userID, roomID); err != nil {
		return nil, err
	}

	msg, err := s.repo.FindByID(messageID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.ErrRecordNotFound
		}
		return nil, err
	}
	if msg.RoomID != roomID {
		return nil, httpx.ErrRecordNotFound
	}

	if pinned && !msg.IsPinned {
		isPinned := true
		count, err := s.repo.Count(ChatMessageFilter{RoomID: roomID, IsPinned: &isPinned})
		if err != nil {
			return nil, err
		}
		if count >= MaxPinnedMessages {
			return nil, httpx.ErrPinnedLimitReached
		}
	}

	if err := s.repo.SetPinned(messageID, pinned); err != nil {
		return nil, err
	}

	msg.IsPinned = pinned
	return msg, nil
}
