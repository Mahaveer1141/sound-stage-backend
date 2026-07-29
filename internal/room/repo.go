package room

import (
	"sound-stage-backend/internal/pkg/listopts"

	"gorm.io/gorm"
)

type Repo interface {
	Create(tx *gorm.DB, input *CreateRoomParams) (*Room, error)
	Update(id uint, input *UpdateRoomParams) (*Room, error)
	FindByID(id uint) (*Room, error)
	List(filter RoomFilter, sort listopts.Sort, p listopts.Pagination) ([]Room, error)
	Count(filter RoomFilter) (int64, error)
}

type repo struct {
	db *gorm.DB
}

func NewRepo(db *gorm.DB) Repo {
	return &repo{db: db}
}

func (r *repo) Create(tx *gorm.DB, input *CreateRoomParams) (*Room, error) {
	room := Room{
		Name:        input.Name,
		Description: input.Description,
		CreatorID:   input.CreatorID,
	}
	if err := tx.Create(&room).Error; err != nil {
		return nil, err
	}
	return &room, nil
}

func (r *repo) List(filter RoomFilter, sort listopts.Sort, p listopts.Pagination) ([]Room, error) {
	var rooms []Room
	err := r.db.
		Preload("Creator").
		Scopes(Filters(filter), Sort(sort), p.Scope()).
		Find(&rooms).Error
	return rooms, err
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

func (r *repo) Count(filter RoomFilter) (int64, error) {
	var count int64
	err := r.db.Model(&Room{}).Scopes(Filters(filter)).Count(&count).Error
	return count, err
}
