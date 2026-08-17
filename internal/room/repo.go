package room

import (
	"sound-stage-backend/internal/pkg/listopts"

	"gorm.io/gorm"
)

type Repo struct {
	db *gorm.DB
}

func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) Create(tx *gorm.DB, input *CreateRoomParams) (*Room, error) {
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

func (r *Repo) List(filter RoomFilter, sort listopts.Sort, p listopts.Pagination) ([]Room, error) {
	var rooms []Room
	err := r.db.
		Preload("Creator").
		Scopes(Filters(filter), Sort(sort), p.Scope()).
		Find(&rooms).Error
	return rooms, err
}

func (r *Repo) FindByID(id uint) (*Room, error) {
	var room Room
	result := r.db.Preload("Creator").Preload("Users").Where("id = ?", id).First(&room)
	if result.Error != nil {
		return nil, result.Error
	}

	return &room, nil
}

func (r *Repo) Update(id uint, input *UpdateRoomParams) (*Room, error) {
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

func (r *Repo) Count(filter RoomFilter) (int64, error) {
	var count int64
	err := r.db.Model(&Room{}).Scopes(Filters(filter)).Count(&count).Error
	return count, err
}
