package roomuser

import "sound-stage-backend/internal/pkg/gormutil"

type Service interface {
	AddUser(userID uint, roomID uint) (*RoomUser, error)
	RemoveUser(userID uint, roomID uint) error
}

type service struct {
	repo Repo
}

func NewService(repo Repo) Service {
	return &service{repo: repo}
}

func (s *service) AddUser(userID uint, roomID uint) (*RoomUser, error) {
	ru, err := s.repo.FindBy(userID, roomID)
	if err != nil {
		return nil, err
	}
	if ru == nil {
		return s.repo.Create(userID, roomID)
	}
	if err := s.repo.UpdateActivity(ru, ActivityJoin); err != nil {
		return nil, err
	}
	return ru, nil
}

func (s *service) RemoveUser(userID uint, roomID uint) error {
	ru, err := s.repo.FindBy(userID, roomID)
	if err != nil {
		return err
	}
	if ru == nil {
		return gormutil.ErrRecordNotFound
	}
	return s.repo.UpdateActivity(ru, ActivityLeave)
}
