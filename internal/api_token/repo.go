package apitoken

import (
	"database/sql"

	"gorm.io/gorm"
)

type Repo struct {
	db *gorm.DB
}

type CreateAPITokenInput struct {
	Token  string
	Type   TokenType
	UserID uint
}

func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) FindByToken(token string) (*APIToken, error) {
	var at APIToken
	result := r.db.
		Where("token = @token AND is_active = @is_active",
			sql.Named("token", token),
			sql.Named("is_active", true)).
		First(&at)

	if result.Error != nil {
		return nil, result.Error
	}

	return &at, nil
}

func (r *Repo) CreateToken(input CreateAPITokenInput) (*APIToken, error) {
	at := APIToken{
		Token:    input.Token,
		Type:     string(input.Type),
		UserID:   input.UserID,
		IsActive: true,
	}
	if err := r.db.Create(&at).Error; err != nil {
		return nil, err
	}

	return &at, nil
}

func (r *Repo) Deactivate(userID uint) error {
	return r.db.Model(&APIToken{}).
		Where("user_id = ? AND is_active = ?", userID, true).
		Update("is_active", false).Error
}
