package category

import (
	model "sound-stage-backend/internal/model"
)

type Category struct {
	model.BaseModel
	Name        string `gorm:"type:citext;not null;uniqueIndex"`
	Description *string
}

type CategoryResponse struct {
	ID          uint    `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

func (Category) TableName() string {
	return "categories"
}
