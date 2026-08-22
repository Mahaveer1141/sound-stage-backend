package tag

import (
	"sound-stage-backend/internal/pkg/listopts"
)

type repository interface {
	Create(input *CreateTagParams) (*Tag, error)
	List(filter TagFilter, sort listopts.Sort, p listopts.Pagination) ([]Tag, error)
	Count(filter TagFilter) (int64, error)
}

type Service struct {
	repo repository
}

func NewService(r repository) *Service {
	return &Service{repo: r}
}

func (s *Service) Create(input *CreateTagParams) (*Tag, error) {
	return s.repo.Create(input)
}

func (s *Service) List(filter TagFilter, sort listopts.Sort, p listopts.Pagination) ([]Tag, int64, error) {
	tags, err := s.repo.List(filter, sort, p)
	if err != nil {
		return nil, 0, err
	}
	count, err := s.repo.Count(filter)
	if err != nil {
		return nil, 0, err
	}
	return tags, count, nil
}
