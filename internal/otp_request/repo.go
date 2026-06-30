package otprequest

import (
	"database/sql"
	"strings"
	"time"

	"gorm.io/gorm"
)

type Repo interface {
	FindByEmail(email string) (*OTPRequest, error)
	Create(otpInput CreateOTPRequestInput) (*OTPRequest, error)
	Deactivate(id uint) error
}

type repo struct {
	db *gorm.DB
}

type CreateOTPRequestInput struct {
	Email  *string
	UserID *uint
	OTP    string
}

func NewRepo(db *gorm.DB) Repo {
	return &repo{db: db}
}

func (r *repo) FindByEmail(email string) (*OTPRequest, error) {
	var otpRequest OTPRequest
	result := r.db.Joins("User").
		Where("(otp_requests.email = @email or \"User\".email = @email) AND otp_requests.is_active = @is_active",
			sql.Named("email", strings.ToLower(email)),
			sql.Named("is_active", true)).
		Order("otp_requests.created_at DESC").
		First(&otpRequest)

	if result.Error != nil {
		return nil, result.Error
	}

	return &otpRequest, nil
}

func (r *repo) Create(otpInput CreateOTPRequestInput) (*OTPRequest, error) {
	var otpRequest OTPRequest

	err := r.db.Transaction(func(tx *gorm.DB) error {
		query := tx.Model(&OTPRequest{}).Where("is_active = ?", true)

		if otpInput.Email != nil {
			query = query.Where("email = ?", strings.ToLower(*otpInput.Email))
		} else if otpInput.UserID != nil {
			query = query.Where("user_id = ?", *otpInput.UserID)
		}

		if err := query.Update("is_active", false).Error; err != nil {
			return err
		}

		otpRequest = OTPRequest{
			Email:     otpInput.Email,
			UserID:    otpInput.UserID,
			OTP:       otpInput.OTP,
			ExpiresAt: time.Now().Add(10 * time.Minute),
			IsActive:  true,
		}

		if err := tx.Create(&otpRequest).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &otpRequest, nil
}

func (r *repo) Deactivate(id uint) error {
	return r.db.Model(&OTPRequest{}).Where("id = ?", id).Update("is_active", false).Error
}
