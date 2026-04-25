package user

import (
	model "sound-stage-backend/internal/model"
	"time"

	"gorm.io/gorm"
)

type User struct {
	model.BaseModel
	Email       string     `gorm:"not null;uniqueIndex" json:"email" validate:"required,email,max=255"`
	FirstName   string     `gorm:"not null" json:"firstName" validate:"required,min=1,max=255"`
	LastName    *string    `json:"lastName,omitempty" validate:"omitempty,max=255"`
	LastLoginAt *time.Time `json:"lastLoginAt,omitempty" validate:"omitempty"`
	DeletedAt   gorm.DeletedAt
}

func (User) TableName() string {
	return "users"
}

type CreateUserParams struct {
	Email     string `json:"email" validate:"required,email"`
	FirstName string `json:"firstName" validate:"required"`
	LastName  string `json:"lastName,omitempty"`
}

type UpdateUserParams struct {
	FirstName string `json:"firstName" validate:"required"`
	LastName  string `json:"lastName,omitempty"`
}
