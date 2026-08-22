package tagging

import (
	model "sound-stage-backend/internal/model"
	tag "sound-stage-backend/internal/tag"
)

type Tagging struct {
	model.BaseModel
	TagID        uint    `gorm:"not null;uniqueIndex:idx_taggables_tag_taggable"`
	TaggableType string  `gorm:"not null;uniqueIndex:idx_taggables_tag_taggable"`
	TaggableID   uint    `gorm:"not null;uniqueIndex:idx_taggables_tag_taggable"`
	Tag          tag.Tag `gorm:"foreignKey:TagID"`
}

func (Tagging) TableName() string {
	return "taggables"
}
