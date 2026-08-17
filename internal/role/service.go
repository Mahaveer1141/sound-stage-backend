package role

type repository interface {
	FindByName(name RoleName) (*Role, error)
}

type Service struct {
	repo repository
}

func NewService(r repository) *Service {
	return &Service{repo: r}
}

func (s *Service) FindByName(name RoleName) (*Role, error) {
	return s.repo.FindByName(name)
}
