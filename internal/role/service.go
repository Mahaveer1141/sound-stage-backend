package role

type Service interface {
	FindByName(name RoleName) (*Role, error)
}

type service struct {
	repo Repo
}

func NewService(repo Repo) Service {
	return &service{repo: repo}
}

func (s *service) FindByName(name RoleName) (*Role, error) {
	return s.repo.FindByName(name)
}
