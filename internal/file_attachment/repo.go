package fileattachment

import (
	"gorm.io/gorm"
)

type Repo struct {
	db *gorm.DB
}

func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) Create(att *FileAttachment) error {
	return r.db.Create(att).Error
}

func (r *Repo) FindByID(id uint) (*FileAttachment, error) {
	var att FileAttachment
	result := r.db.First(&att, id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &att, nil
}

func (r *Repo) Save(att *FileAttachment) error {
	return r.db.Save(att).Error
}

func (r *Repo) Delete(att *FileAttachment) error {
	return r.db.Delete(att).Error
}
