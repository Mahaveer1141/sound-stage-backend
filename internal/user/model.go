package user

import (
	model "sound-stage-backend/internal/model"
	"time"

	"gorm.io/gorm"
)

type User struct {
	model.BaseModel
	Email       string     `gorm:"not null;uniqueIndex" validate:"required,email,max=255"`
	FirstName   string     `gorm:"not null" validate:"required,min=1,max=255"`
	LastName    *string    `validate:"omitempty,max=255"`
	LastLoginAt *time.Time `validate:"omitempty"`
	DeletedAt   gorm.DeletedAt
}

func (User) TableName() string {
	return "users"
}

type UserResponse struct {
	ID        uint    `json:"id"`
	Email     string  `json:"email,omitempty"`
	FirstName string  `json:"firstName"`
	LastName  *string `json:"lastName,omitempty"`
	FullName  string  `json:"fullName"`
}

func (u *User) FullName() string {
	if u.LastName == nil {
		return u.FirstName
	}
	return u.FirstName + " " + *u.LastName
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
