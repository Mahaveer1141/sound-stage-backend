package apitoken

import (
	"sound-stage-backend/internal/model"
	"time"
)

type APIToken struct {
	model.BaseModel
	Token     string    `validate:"required"`
	Type      string    `validate:"required,oneof=access refresh"`
	IsActive  bool      `validate:"required"`
	UserID    uint      `validate:"required"`
	ExpiresAt time.Time `validate:"required"`
}

func (APIToken) TableName() string {
	return "api_tokens"
}
