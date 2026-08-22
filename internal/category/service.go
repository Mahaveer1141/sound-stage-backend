package category

type repository interface {
	List() ([]Category, error)
}

type Service struct {
	repo repository
}

func NewService(r repository) *Service {
	return &Service{repo: r}
}

func (s *Service) List() ([]Category, error) {
	categories, err := s.repo.List()
	if err != nil {
		return nil, err
	}
	return categories, nil
}
