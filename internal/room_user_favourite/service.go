package roomuserfavourite

type repo interface {
	Add(userID, roomID uint) error
	Remove(userID, roomID uint) error
	FindRoomIDsByUserID(userID uint, roomIDs []uint) ([]uint, error)
}

type authorizer interface {
	CanAdd(userID, roomID uint) error
}

type Service struct {
	repo  repo
	authz authorizer
}

func NewService(r repo, authz authorizer) *Service {
	return &Service{repo: r, authz: authz}
}

func (s *Service) Add(userID, roomID uint) error {
	if err := s.authz.CanAdd(userID, roomID); err != nil {
		return err
	}
	return s.repo.Add(userID, roomID)
}

func (s *Service) Remove(userID, roomID uint) error {
	return s.repo.Remove(userID, roomID)
}

func (s *Service) FavouritedRoomIDs(userID uint, roomIDs []uint) (map[uint]bool, error) {
	ids, err := s.repo.FindRoomIDsByUserID(userID, roomIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[uint]bool, len(ids))
	for _, id := range ids {
		out[id] = true
	}
	return out, nil
}
