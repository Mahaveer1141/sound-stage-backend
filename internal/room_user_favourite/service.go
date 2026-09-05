package roomuserfavourite

import (
	"sound-stage-backend/internal/pkg/listopts"
)

type repo interface {
	Add(userID, roomID uint) error
	Remove(userID, roomID uint) error
	ListUserFavourites(userID uint, p listopts.Pagination) ([]RoomUserFavourite, error)
	CountByUserID(userID uint) (int64, error)
}

type Service struct {
	repo repo
}

func NewService(r repo) *Service {
	return &Service{repo: r}
}

func (s *Service) Add(userID, roomID uint) error {
	return s.repo.Add(userID, roomID)
}

func (s *Service) Remove(userID, roomID uint) error {
	return s.repo.Remove(userID, roomID)
}

func (s *Service) ListUserFavourites(userID uint, p listopts.Pagination) ([]RoomUserFavourite, int64, error) {
	favourites, err := s.repo.ListUserFavourites(userID, p)
	if err != nil {
		return nil, 0, err
	}

	count, err := s.repo.CountByUserID(userID)
	if err != nil {
		return nil, 0, err
	}

	return favourites, count, nil
}
