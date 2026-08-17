package user

import (
	"sound-stage-backend/internal/pkg/gormutil"
	"strings"
	"time"

	"gorm.io/gorm"
)

type Repo struct {
	db *gorm.DB
}

func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) Create(input *CreateUserParams) (*User, error) {
	user := User{
		Email:     input.Email,
		FirstName: input.FirstName,
		LastName:  &input.LastName,
	}
	if err := r.db.Create(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repo) FindByEmail(email string) (*User, error) {
	var user User
	result := r.db.Where("email = ?", strings.ToLower(email)).First(&user)
	return gormutil.NilIfNotFound(&user, result.Error)
}

func (r *Repo) FindByID(id uint) (*User, error) {
	var user User
	result := r.db.Where("id = ?", id).First(&user)
	return gormutil.NilIfNotFound(&user, result.Error)
}

func (r *Repo) UpdateLastLoginAt(id uint) error {
	return r.db.Model(&User{}).Where("id = ?", id).Update("last_login_at", time.Now()).Error
}

func (r *Repo) Update(id uint, input *UpdateUserParams) (*User, error) {
	var user User
	result := r.db.Where("id = ?", id).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}

	user.FirstName = input.FirstName
	user.LastName = &input.LastName

	if err := r.db.Save(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}
