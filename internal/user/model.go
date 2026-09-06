package user

import (
	"mime/multipart"
	fileattachment "sound-stage-backend/internal/file_attachment"
	model "sound-stage-backend/internal/model"
	"sound-stage-backend/internal/pkg/httpx"
	"time"

	"gorm.io/gorm"
)

type User struct {
	model.BaseModel
	Email          string                         `gorm:"not null;uniqueIndex" validate:"required,email,max=255"`
	FirstName      string                         `gorm:"not null" validate:"required,min=1,max=255"`
	LastName       *string                        `validate:"omitempty,max=255"`
	LastLoginAt    *time.Time                     `validate:"omitempty"`
	ProfilePicture *fileattachment.FileAttachment `gorm:"polymorphic:Owner;"`
	DeletedAt      gorm.DeletedAt
}

func (User) TableName() string {
	return "users"
}

type UserResponse struct {
	ID             uint                          `json:"id"`
	Email          string                        `json:"email,omitempty"`
	FirstName      string                        `json:"firstName"`
	LastName       *string                       `json:"lastName,omitempty"`
	FullName       string                        `json:"fullName"`
	ProfilePicture *httpx.FileAttachmentResponse `json:"profilePicture,omitempty"`
}

func (u *User) FullName() string {
	if u.LastName == nil {
		return u.FirstName
	}
	return u.FirstName + " " + *u.LastName
}

type CreateUserParams struct {
	Email          string                `form:"email" json:"email" validate:"required,email"`
	FirstName      string                `form:"firstName" json:"firstName" validate:"required"`
	LastName       string                `form:"lastName" json:"lastName"`
	ProfilePicture *multipart.FileHeader `form:"profilePicture" json:"-" validate:"omitempty,max_size=10485760"`
}

type UpdateUserParams struct {
	FirstName      string                `form:"firstName" json:"firstName" validate:"required"`
	LastName       string                `form:"lastName" json:"lastName"`
	ProfilePicture *multipart.FileHeader `form:"profilePicture" json:"-" validate:"omitempty,max_size=10485760"`
}
