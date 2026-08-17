package otprequest

type repository interface {
	FindByEmail(email string) (*OTPRequest, error)
	Create(otpInput CreateOTPRequestInput) (*OTPRequest, error)
	Deactivate(id uint) error
}

type Service struct {
	repo repository
}

func NewService(r repository) *Service {
	return &Service{repo: r}
}

func (s *Service) FindByEmail(email string) (*OTPRequest, error) {
	return s.repo.FindByEmail(email)
}

func (s *Service) Create(input CreateOTPRequestInput) (*OTPRequest, error) {
	return s.repo.Create(input)
}

func (s *Service) Deactivate(id uint) error {
	return s.repo.Deactivate(id)
}
