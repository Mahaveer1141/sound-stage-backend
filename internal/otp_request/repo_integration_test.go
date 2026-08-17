package otprequest

import (
	"testing"
	"time"

	"sound-stage-backend/internal/model"
	"sound-stage-backend/internal/pkg/httpx"
	"sound-stage-backend/internal/pkg/testutil"
	"sound-stage-backend/internal/user"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func seedOTPRequests(t *testing.T, db *gorm.DB, reqs ...OTPRequest) {
	t.Helper()
	for i := range reqs {
		require.NoError(t, db.Create(&reqs[i]).Error)
	}
}

func TestRepo_FindByEmail_Integration(t *testing.T) {
	t.Run("finds the most recent active request by email, case-insensitively", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &OTPRequest{}, &user.User{})
		seedOTPRequests(t, db,
			OTPRequest{BaseModel: model.BaseModel{CreatedAt: time.Now().Add(-2 * time.Minute)}, Email: strPtr("user@example.com"), OTP: "111111", ExpiresAt: time.Now().Add(10 * time.Minute), IsActive: false},
			OTPRequest{Email: strPtr("user@example.com"), OTP: "222222", ExpiresAt: time.Now().Add(20 * time.Minute), IsActive: true},
		)
		repo := NewRepo(db)

		got, err := repo.FindByEmail("USER@Example.com")

		require.NoError(t, err)
		require.NotNil(t, got)
		require.Equal(t, "222222", got.OTP)
		require.True(t, got.IsActive)
	})

	t.Run("finds active request by associated user email", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &OTPRequest{}, &user.User{})

		u := user.User{Email: "user@example.com", FirstName: "Test"}
		require.NoError(t, db.Create(&u).Error)

		seedOTPRequests(t, db,
			OTPRequest{UserID: &u.ID, OTP: "555555", ExpiresAt: time.Now().Add(10 * time.Minute), IsActive: true},
		)
		repo := NewRepo(db)

		got, err := repo.FindByEmail("USER@Example.com")

		require.NoError(t, err)
		require.NotNil(t, got)
		require.Equal(t, "555555", got.OTP)
		require.True(t, got.IsActive)
	})

	t.Run("returns error when no active request exists", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &OTPRequest{}, &user.User{})
		seedOTPRequests(t, db,
			OTPRequest{Email: strPtr("user@example.com"), OTP: "111111", ExpiresAt: time.Now().Add(10 * time.Minute), IsActive: false},
		)
		repo := NewRepo(db)

		got, err := repo.FindByEmail("user@example.com")

		require.Error(t, err)
		require.Nil(t, got)
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})
}

func TestRepo_Create_Integration(t *testing.T) {
	t.Run("deactivates prior active requests for the same email and creates a new one", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &OTPRequest{}, &user.User{})
		repo := NewRepo(db)

		first := OTPRequest{
			BaseModel: model.BaseModel{CreatedAt: time.Now().Add(-2 * time.Minute)},
			Email:     strPtr("user@example.com"),
			OTP:       "111111",
			ExpiresAt: time.Now().Add(10 * time.Minute),
			IsActive:  true,
		}
		require.NoError(t, db.Create(&first).Error)

		second, err := repo.Create(CreateOTPRequestInput{Email: strPtr("user@example.com"), OTP: "222222"})
		require.NoError(t, err)
		require.True(t, second.IsActive)

		var reloadedFirst OTPRequest
		require.NoError(t, db.First(&reloadedFirst, first.ID).Error)
		require.False(t, reloadedFirst.IsActive)

		var reloadedSecond OTPRequest
		require.NoError(t, db.First(&reloadedSecond, second.ID).Error)
		require.True(t, reloadedSecond.IsActive)
	})

	t.Run("deactivates prior active requests for the same user_id and creates a new one", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &OTPRequest{}, &user.User{})
		repo := NewRepo(db)

		first := OTPRequest{
			BaseModel: model.BaseModel{CreatedAt: time.Now().Add(-2 * time.Minute)},
			UserID:    uintPtr(5),
			OTP:       "111111",
			ExpiresAt: time.Now().Add(10 * time.Minute),
			IsActive:  true,
		}
		require.NoError(t, db.Create(&first).Error)

		second, err := repo.Create(CreateOTPRequestInput{UserID: uintPtr(5), OTP: "222222"})
		require.NoError(t, err)
		require.True(t, second.IsActive)

		var reloadedFirst OTPRequest
		require.NoError(t, db.First(&reloadedFirst, first.ID).Error)
		require.False(t, reloadedFirst.IsActive)

		var reloadedSecond OTPRequest
		require.NoError(t, db.First(&reloadedSecond, second.ID).Error)
		require.True(t, reloadedSecond.IsActive)
	})

	t.Run("does not deactivate requests for a different email", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &OTPRequest{}, &user.User{})
		repo := NewRepo(db)

		other, err := repo.Create(CreateOTPRequestInput{Email: strPtr("other@example.com"), OTP: "333333"})
		require.NoError(t, err)

		_, err = repo.Create(CreateOTPRequestInput{Email: strPtr("user@example.com"), OTP: "444444"})
		require.NoError(t, err)

		var reloadedOther OTPRequest
		require.NoError(t, db.First(&reloadedOther, other.ID).Error)
		require.True(t, reloadedOther.IsActive)
	})

	t.Run("does not deactivate requests for a different user_id", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &OTPRequest{}, &user.User{})
		repo := NewRepo(db)

		other, err := repo.Create(CreateOTPRequestInput{UserID: uintPtr(5), OTP: "333333"})
		require.NoError(t, err)

		_, err = repo.Create(CreateOTPRequestInput{UserID: uintPtr(6), OTP: "444444"})
		require.NoError(t, err)

		var reloadedOther OTPRequest
		require.NoError(t, db.First(&reloadedOther, other.ID).Error)
		require.True(t, reloadedOther.IsActive)
	})

	t.Run("returns error when a recent request already exists for the same email", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &OTPRequest{}, &user.User{})
		repo := NewRepo(db)

		_, err := repo.Create(CreateOTPRequestInput{Email: strPtr("user@example.com"), OTP: "111111"})
		require.NoError(t, err)

		got, err := repo.Create(CreateOTPRequestInput{Email: strPtr("user@example.com"), OTP: "222222"})
		require.Error(t, err)
		require.Nil(t, got)
		require.ErrorIs(t, err, httpx.ErrOTPRequestAlreadyMade)
	})

	t.Run("returns error when a recent request already exists for the same user_id", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &OTPRequest{}, &user.User{})
		repo := NewRepo(db)

		_, err := repo.Create(CreateOTPRequestInput{UserID: uintPtr(5), OTP: "111111"})
		require.NoError(t, err)

		got, err := repo.Create(CreateOTPRequestInput{UserID: uintPtr(5), OTP: "222222"})
		require.Error(t, err)
		require.Nil(t, got)
		require.ErrorIs(t, err, httpx.ErrOTPRequestAlreadyMade)
	})

	t.Run("returns error when neither email nor user_id is provided", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &OTPRequest{}, &user.User{})
		repo := NewRepo(db)

		got, err := repo.Create(CreateOTPRequestInput{OTP: "111111"})
		require.Error(t, err)
		require.Nil(t, got)
		require.ErrorIs(t, err, httpx.ErrUserOrEmailRequired)
	})
}

func TestRepo_Deactivate_Integration(t *testing.T) {
	t.Run("deactivates the given request only", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &OTPRequest{}, &user.User{})
		repo := NewRepo(db)

		target, err := repo.Create(CreateOTPRequestInput{Email: strPtr("a@example.com"), OTP: "111111"})
		require.NoError(t, err)
		untouched, err := repo.Create(CreateOTPRequestInput{Email: strPtr("b@example.com"), OTP: "222222"})
		require.NoError(t, err)

		require.NoError(t, repo.Deactivate(target.ID))

		var reloadedTarget OTPRequest
		require.NoError(t, db.First(&reloadedTarget, target.ID).Error)
		require.False(t, reloadedTarget.IsActive)

		var reloadedUntouched OTPRequest
		require.NoError(t, db.First(&reloadedUntouched, untouched.ID).Error)
		require.True(t, reloadedUntouched.IsActive)
	})
}
