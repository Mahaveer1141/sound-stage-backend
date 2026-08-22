package tag

import (
	"strings"

	"sound-stage-backend/internal/pkg/listopts"

	"gorm.io/gorm"
)

type Repo struct {
	db *gorm.DB
}

func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) Create(input *CreateTagParams) (*Tag, error) {
	tag := Tag{
		Name: strings.TrimSpace(input.Name),
	}
	if err := r.db.Create(&tag).Error; err != nil {
		return nil, err
	}
	return &tag, nil
}

func (r *Repo) List(filter TagFilter, sort listopts.Sort, p listopts.Pagination) ([]Tag, error) {
	var tags []Tag
	err := r.db.
		Scopes(Filters(filter), Sort(sort), p.Scope()).
		Find(&tags).Error
	return tags, err
}

func (r *Repo) Count(filter TagFilter) (int64, error) {
	var count int64
	err := r.db.Model(&Tag{}).Scopes(Filters(filter)).Count(&count).Error
	return count, err
}
