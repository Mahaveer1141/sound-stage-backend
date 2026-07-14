package role

type Service interface {
	FindByName(name string) (*Role, error)
}

type service struct {
	repo Repo
}

func NewService(repo Repo) Service {
	return &service{repo: repo}
}

func (s *service) FindByName(name string) (*Role, error) {
	return s.repo.FindByName(name)
}
