package user

type Service interface {
	FindByID(userId uint) (*User, error)
	FindByEmail(email string) (*User, error)
	Create(input *CreateUserParams) (*User, error)
	UpdateLastLoginAt(id uint) error
	UpdateProfile(id uint, input *UpdateUserParams) (*User, error)
}

type service struct {
	repo Repo
}

func NewService(repo Repo) Service {
	return &service{repo: repo}
}

func (s *service) FindByID(userId uint) (*User, error) {
	return s.repo.FindByID(userId)
}

func (s *service) FindByEmail(email string) (*User, error) {
	return s.repo.FindByEmail(email)
}

func (s *service) Create(input *CreateUserParams) (*User, error) {
	return s.repo.Create(input)
}

func (s *service) UpdateLastLoginAt(id uint) error {
	return s.repo.UpdateLastLoginAt(id)
}

func (s *service) UpdateProfile(id uint, input *UpdateUserParams) (*User, error) {
	return s.repo.Update(id, input)
}
