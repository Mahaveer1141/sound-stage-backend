package roomuserfavourite

type repo interface {
	Add(userID, roomID uint) error
	Remove(userID, roomID uint) error
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
