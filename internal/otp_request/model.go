package otprequest

import (
	"errors"
	"sound-stage-backend/internal/model"
	"sound-stage-backend/internal/user"
	"time"

	"gorm.io/gorm"
)

type OTPRequest struct {
	model.BaseModel
	Email     *string `validate:"email"`
	OTP       string  `gorm:"not null" validate:"required"`
	UserID    *uint
	ExpiresAt time.Time  `gorm:"not null" validate:"required"`
	IsActive  bool       `gorm:"not null" validate:"required"`
	User      *user.User `gorm:"foreignKey:UserID" `
}

func (OTPRequest) TableName() string {
	return "otp_requests"
}

func (or *OTPRequest) IsExpired() bool {
	return or.ExpiresAt.Before(time.Now().UTC())
}

func (or *OTPRequest) VerifyOTP(otp string) bool {
	return !or.IsExpired() && or.IsActive
}

func (or *OTPRequest) BeforeCreate(tx *gorm.DB) error {
	oneMinuteAgo := time.Now().Add(-1 * time.Minute)

	query := tx.Where("created_at > ?", oneMinuteAgo)

	if or.UserID != nil {
		query = query.Where("user_id = ?", or.UserID)
	} else if or.Email != nil {
		query = query.Where("email = ?", or.Email)
	} else {
		return errors.New("either user_id or email must be provided")
	}

	var count int64
	if err := query.Model(&OTPRequest{}).Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		return errors.New("OTP request already made within the last minute, please wait")
	}

	return nil
}
