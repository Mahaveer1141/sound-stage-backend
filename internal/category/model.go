package category

import (
	model "sound-stage-backend/internal/model"
)

type Category struct {
	model.BaseModel
	Name        string  `gorm:"type:citext;not null;uniqueIndex" json:"name" validate:"required"`
	Description *string `json:"description,omitempty" validate:"omitempty"`
}

func (Category) TableName() string {
	return "categories"
}
