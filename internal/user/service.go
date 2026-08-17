package user

type repository interface {
	Create(input *CreateUserParams) (*User, error)
	FindByID(id uint) (*User, error)
	FindByEmail(email string) (*User, error)
	UpdateLastLoginAt(id uint) error
	Update(id uint, input *UpdateUserParams) (*User, error)
}

type Service struct {
	repo repository
}

func NewService(r repository) *Service {
	return &Service{repo: r}
}

func (s *Service) FindByID(userId uint) (*User, error) {
	return s.repo.FindByID(userId)
}

func (s *Service) FindByEmail(email string) (*User, error) {
	return s.repo.FindByEmail(email)
}

func (s *Service) Create(input *CreateUserParams) (*User, error) {
	return s.repo.Create(input)
}

func (s *Service) UpdateLastLoginAt(id uint) error {
	return s.repo.UpdateLastLoginAt(id)
}

func (s *Service) UpdateProfile(id uint, input *UpdateUserParams) (*User, error) {
	return s.repo.Update(id, input)
}
