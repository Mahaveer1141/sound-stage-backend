package room

import (
	"sound-stage-backend/internal/category"
	"sound-stage-backend/internal/pkg/listopts"
	roomcategory "sound-stage-backend/internal/room_category"
	"sound-stage-backend/internal/tag"
	"sound-stage-backend/internal/tagging"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	if err := addCategories(tx, &room, input.CategoryIds, true); err != nil {
		return nil, err
	}
	if err := addTags(tx, &room, input.TagIds, true); err != nil {
		return nil, err
	}
	return &room, nil
}

func (r *Repo) List(filter RoomFilter, sort listopts.Sort, p listopts.Pagination) ([]Room, error) {
	var rooms []Room
	err := r.db.Preload("Creator").Preload("Categories").
		Scopes(Filters(filter), Sort(sort), p.Scope()).
		Find(&rooms).Error
	return rooms, err
}

func (r *Repo) FindByID(id uint) (*Room, error) {
	var room Room
	result := r.db.Preload("Creator").Where("id = ?", id).First(&room)
	if result.Error != nil {
		return nil, result.Error
	}

	return &room, nil
}

func (r *Repo) Update(id uint, input *UpdateRoomParams) (*Room, error) {
	var room Room
	result := r.db.First(&room, id)
	if result.Error != nil {
		return nil, result.Error
	}

	var categories []category.Category
	if err := r.db.Where("id IN ?", input.CategoryIds).Find(&categories).Error; err != nil {
		return nil, err
	}

	err := r.db.Transaction(func(tx *gorm.DB) error {
		room.Name = input.Name
		room.Description = input.Description
		if err := tx.Save(&room).Error; err != nil {
			return err
		}

		if err := addCategories(tx, &room, input.CategoryIds, false); err != nil {
			return err
		}
		if err := addTags(tx, &room, input.TagIds, false); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &room, nil
}

func (r *Repo) Count(filter RoomFilter) (int64, error) {
	var count int64
	err := r.db.Model(&Room{}).Scopes(Filters(filter)).Count(&count).Error
	return count, err
}

func (r *Repo) LoadTagsForRooms(roomIDs []uint) (map[uint][]tag.Tag, error) {
	type row struct {
		tag.Tag
		RoomID uint
	}
	var rows []row
	err := r.db.Table("tags").
		Select("tags.*, taggables.taggable_id as room_id").
		Joins("JOIN taggables ON taggables.tag_id = tags.id").
		Where("taggables.taggable_type = ? AND taggables.taggable_id IN ?", "Room", roomIDs).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[uint][]tag.Tag)
	for _, row := range rows {
		out[row.RoomID] = append(out[row.RoomID], row.Tag)
	}
	return out, nil
}

func addCategories(tx *gorm.DB, room *Room, categoryIds []uint, isCreate bool) error {
	deleteQuery := tx.Where("room_id = ?", room.ID)
	if !isCreate {
		if len(categoryIds) > 0 {
			deleteQuery = deleteQuery.Where("category_id NOT IN ?", categoryIds)
		}
		if err := deleteQuery.Delete(&roomcategory.RoomCategory{}).Error; err != nil {
			return err
		}
	}
	if len(categoryIds) > 0 {
		var roomCategories []roomcategory.RoomCategory
		for _, id := range categoryIds {
			roomCategories = append(roomCategories, roomcategory.RoomCategory{RoomID: room.ID, CategoryID: id})
		}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&roomCategories).Error; err != nil {
			return err
		}
	}
	return nil
}

func addTags(tx *gorm.DB, room *Room, tagIds []uint, isCreate bool) error {
	deleteQuery := tx.Where("taggable_type = 'Room' AND taggable_id = ?", room.ID)
	if !isCreate {
		if len(tagIds) > 0 {
			deleteQuery = deleteQuery.Where("tag_id NOT IN ?", tagIds)
		}
		if err := deleteQuery.Delete(&tagging.Tagging{}).Error; err != nil {
			return err
		}
	}
	if len(tagIds) > 0 {
		var roomTags []tagging.Tagging
		for _, id := range tagIds {
			roomTags = append(roomTags, tagging.Tagging{TaggableType: "Room", TaggableID: room.ID, TagID: id})
		}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&roomTags).Error; err != nil {
			return err
		}
	}
	return nil
}
