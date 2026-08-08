package role

import (
	"gorm.io/gorm"
)

type Repo interface {
	FindByName(name RoleName) (*Role, error)
}

type repo struct {
	db *gorm.DB
}

func NewRepo(db *gorm.DB) Repo {
	return &repo{db: db}
}

func (r *repo) FindByName(name RoleName) (*Role, error) {
	var role Role
	result := r.db.Where("name = ?", name).First(&role)
	if result.Error != nil {
		return nil, result.Error
	}
	return &role, nil
}
