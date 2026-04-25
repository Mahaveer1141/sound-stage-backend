package otprequest

type Service interface {
	FindByEmail(email string) (*OTPRequest, error)
	Create(input CreateOTPRequestInput) (*OTPRequest, error)
	Deactivate(id uint) error
}

type service struct {
	repo Repo
}

func NewService(repo Repo) Service {
	return &service{repo: repo}
}

func (s *service) FindByEmail(email string) (*OTPRequest, error) {
	return s.repo.FindByEmail(email)
}

func (s *service) Create(input CreateOTPRequestInput) (*OTPRequest, error) {
	return s.repo.Create(input)
}

func (s *service) Deactivate(id uint) error {
	return s.repo.Deactivate(id)
}
