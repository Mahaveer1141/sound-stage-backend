package category

import (
	"gorm.io/gorm"
)

type Repo struct {
	db *gorm.DB
}

func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) List() ([]Category, error) {
	var categories []Category
	err := r.db.Find(&categories).Error
	return categories, err
}
