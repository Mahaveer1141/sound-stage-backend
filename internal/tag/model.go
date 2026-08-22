package tag

import (
	"strings"

	model "sound-stage-backend/internal/model"
	"sound-stage-backend/internal/pkg/listopts"

	"gorm.io/gorm"
)

type Tag struct {
	model.BaseModel
	Name string `gorm:"type:citext;not null;uniqueIndex" json:"name" validate:"required"`
}

func (Tag) TableName() string {
	return "tags"
}

type CreateTagParams struct {
	Name string `json:"name" validate:"required"`
}

type TagFilter struct {
	Query string `form:"query"`
}

var allowedSortFields = map[string]string{
	"name":       "tags.name",
	"created_at": "tags.created_at",
}

func FilterBySearch(query string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if query == "" {
			return db
		}
		return db.Where("tags.name LIKE ?", "%"+query+"%")
	}
}

func Filters(f TagFilter) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Scopes(
			FilterBySearch(f.Query),
		)
	}
}

func Sort(s listopts.Sort) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		col, ok := allowedSortFields[strings.ToLower(s.Field)]
		if !ok {
			col = "tags.created_at"
		}

		order := strings.ToLower(s.Order)
		if order != "asc" && order != "desc" {
			order = "desc"
		}
		return db.Order(col + " " + order)
	}
}
