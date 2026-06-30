package room

import (
	"gorm.io/gorm"
)

type Repo interface {
	Create(input *CreateRoomParams) (*Room, error)
	Update(id uint, input *UpdateRoomParams) (*Room, error)
	FindByID(id uint) (*Room, error)
	List(page, pageSize int) ([]Room, error)
	Count() (int, error)
}

type repo struct {
	db *gorm.DB
}

func NewRepo(db *gorm.DB) Repo {
	return &repo{db: db}
}

func (r *repo) Create(input *CreateRoomParams) (*Room, error) {
	room := Room{
		Name:        input.Name,
		Description: input.Description,
		CreatorID:   input.CreatorID,
	}
	if err := r.db.Create(&room).Error; err != nil {
		return nil, err
	}
	return &room, nil
}

func (r *repo) List(page, pageSize int) ([]Room, error) {
	var rooms []Room
	offset := (page - 1) * pageSize

	err := r.db.Preload("Creator").Limit(pageSize).Offset(offset).Find(&rooms).Error

	if err != nil {
		return nil, err
	}
	return rooms, nil
}

func (r *repo) FindByID(id uint) (*Room, error) {
	var room Room
	result := r.db.Preload("Creator").Preload("Users").Where("id = ?", id).First(&room)
	if result.Error != nil {
		return nil, result.Error
	}

	return &room, nil
}

func (r *repo) Update(id uint, input *UpdateRoomParams) (*Room, error) {
	var room Room
	result := r.db.Where("id = ?", id).First(&room)
	if result.Error != nil {
		return nil, result.Error
	}

	room.Name = input.Name
	room.Description = input.Description
	if err := r.db.Save(&room).Error; err != nil {
		return nil, err
	}
	return &room, nil
}

func (r *repo) Count() (int, error) {
	var count int64
	err := r.db.Model(&Room{}).Count(&count).Error
	if err != nil {
		return 0, err
	}

	return int(count), nil
}
