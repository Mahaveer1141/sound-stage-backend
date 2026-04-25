package room

type Service interface {
	FindByID(id uint) (*Room, error)
	Create(input *CreateRoomParams) (*Room, error)
	Update(id uint, input *UpdateRoomParams) (*Room, error)
	List(page, pageSize int) ([]Room, error)
	Count() (int, error)
}

type service struct {
	repo Repo
}

func NewService(repo Repo) Service {
	return &service{repo: repo}
}

func (s *service) FindByID(id uint) (*Room, error) {
	return s.repo.FindByID(id)
}

func (s *service) Create(input *CreateRoomParams) (*Room, error) {
	return s.repo.Create(input)
}

func (s *service) Update(id uint, input *UpdateRoomParams) (*Room, error) {
	return s.repo.Update(id, input)
}

func (s *service) List(page, pageSize int) ([]Room, error) {
	return s.repo.List(page, pageSize)
}

func (s *service) Count() (int, error) {
	return s.repo.Count()
}
