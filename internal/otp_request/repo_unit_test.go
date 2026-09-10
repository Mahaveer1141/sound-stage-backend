package otprequest

import (
	"database/sql/driver"
	"regexp"
	"testing"
	"time"

	"sound-stage-backend/internal/pkg/httpx"
	"sound-stage-backend/internal/pkg/testutil"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func strPtr(s string) *string { return &s }
func uintPtr(u uint) *uint    { return &u }

func TestOTPRequest_IsExpired(t *testing.T) {
	t.Run("returns true when expiration time is in the past", func(t *testing.T) {
		req := OTPRequest{ExpiresAt: time.Now().UTC().Add(-5 * time.Minute)}
		assert.True(t, req.IsExpired())
	})

	t.Run("returns false when expiration time is in the future", func(t *testing.T) {
		req := OTPRequest{ExpiresAt: time.Now().UTC().Add(5 * time.Minute)}
		assert.False(t, req.IsExpired())
	})
}

func TestOTPRequest_VerifyOTP(t *testing.T) {
	t.Run("returns true when OTP matches, request is active, and not expired", func(t *testing.T) {
		req := OTPRequest{
			OTP:       "123456",
			IsActive:  true,
			ExpiresAt: time.Now().UTC().Add(5 * time.Minute),
		}
		assert.True(t, req.VerifyOTP("123456"))
	})

	t.Run("returns false when OTP does not match", func(t *testing.T) {
		req := OTPRequest{
			OTP:       "123456",
			IsActive:  true,
			ExpiresAt: time.Now().UTC().Add(5 * time.Minute),
		}
		assert.False(t, req.VerifyOTP("654321"))
	})

	t.Run("returns false when request is inactive", func(t *testing.T) {
		req := OTPRequest{
			OTP:       "123456",
			IsActive:  false,
			ExpiresAt: time.Now().UTC().Add(5 * time.Minute),
		}
		assert.False(t, req.VerifyOTP("123456"))
	})

	t.Run("returns false when request is expired", func(t *testing.T) {
		req := OTPRequest{
			OTP:       "123456",
			IsActive:  true,
			ExpiresAt: time.Now().UTC().Add(-5 * time.Minute),
		}
		assert.False(t, req.VerifyOTP("123456"))
	})
}

func TestOTPRequest_BeforeCreate(t *testing.T) {
	email := "user@example.com"
	userID := uint(10)

	t.Run("returns error when neither Email nor UserID is set", func(t *testing.T) {
		gdb, _ := testutil.NewMockDB(t)
		req := &OTPRequest{}

		err := req.BeforeCreate(gdb)
		assert.ErrorIs(t, err, httpx.ErrUserOrEmailRequired)
	})

	t.Run("returns error when a recent request exists by UserID even if Email is also set", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		req := &OTPRequest{
			Email:  &email,
			UserID: &userID,
		}

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "otp_requests" WHERE created_at > $1 AND user_id = $2`)).
			WithArgs(sqlmock.AnyArg(), driver.Value(userID)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		err := req.BeforeCreate(gdb)
		assert.ErrorIs(t, err, httpx.ErrOTPRequestAlreadyMade)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when recent OTP request exists by Email", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		req := &OTPRequest{Email: &email}

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "otp_requests" WHERE created_at > $1 AND email = $2`)).
			WithArgs(sqlmock.AnyArg(), driver.Value(email)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		err := req.BeforeCreate(gdb)
		assert.ErrorIs(t, err, httpx.ErrOTPRequestAlreadyMade)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when recent OTP request exists by UserID", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		req := &OTPRequest{UserID: &userID}

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "otp_requests" WHERE created_at > $1 AND user_id = $2`)).
			WithArgs(sqlmock.AnyArg(), driver.Value(userID)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		err := req.BeforeCreate(gdb)
		assert.ErrorIs(t, err, httpx.ErrOTPRequestAlreadyMade)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("passes when no recent OTP request exists", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		req := &OTPRequest{Email: &email}

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "otp_requests" WHERE created_at > $1 AND email = $2`)).
			WithArgs(sqlmock.AnyArg(), driver.Value(email)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		err := req.BeforeCreate(gdb)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns database error when count query fails", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		req := &OTPRequest{Email: &email}

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "otp_requests" WHERE created_at > $1 AND email = $2`)).
			WithArgs(sqlmock.AnyArg(), driver.Value(email)).
			WillReturnError(assert.AnError)

		err := req.BeforeCreate(gdb)
		assert.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepo_FindByEmail_Unit(t *testing.T) {
	cols := []string{"id", "email", "otp", "user_id", "expires_at", "is_active"}

	t.Run("returns active otp request when found", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		email := "user@example.com"
		mock.ExpectQuery(
			`(?s)WHERE \(otp_requests\.email = \$1 or "User"\.email = \$2\) AND otp_requests\.is_active = \$3 ORDER BY otp_requests\.created_at DESC.*LIMIT \$4`,
		).
			WithArgs(email, email, true, 1).
			WillReturnRows(
				sqlmock.NewRows(cols).
					AddRow(1, "user@example.com", "123456", nil, time.Now().Add(time.Hour), true),
			)

		got, err := repo.FindByEmail("USER@example.com")

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, uint(1), got.ID)
		assert.Equal(t, "123456", got.OTP)
		assert.True(t, got.IsActive)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when not found", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(
			`(?s)WHERE \(otp_requests\.email = \$1 or "User"\.email = \$2\) AND otp_requests\.is_active = \$3 ORDER BY otp_requests\.created_at DESC.*LIMIT \$4`,
		).
			WithArgs("missing@example.com", "missing@example.com", true, 1).
			WillReturnError(gorm.ErrRecordNotFound)

		got, err := repo.FindByEmail("missing@example.com")

		require.Error(t, err)
		require.Nil(t, got)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns wrapped database error for other errors", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		email := "user@example.com"
		mock.ExpectQuery(
			`(?s)WHERE \(otp_requests\.email = \$1 or "User"\.email = \$2\) AND otp_requests\.is_active = \$3 ORDER BY otp_requests\.created_at DESC.*LIMIT \$4`,
		).
			WithArgs(email, email, true, 1).
			WillReturnError(assert.AnError)

		got, err := repo.FindByEmail(email)

		require.Error(t, err)
		require.Nil(t, got)
		assert.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepo_Create_Unit(t *testing.T) {
	t.Run("deactivates existing active requests for email then creates new", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "otp_requests" SET "is_active"=$1,"updated_at"=$2 WHERE is_active = $3 AND email = $4`)).
			WithArgs(driver.Value(false), sqlmock.AnyArg(), driver.Value(true), driver.Value("user@example.com")).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT count(*) FROM "otp_requests" WHERE created_at > $1 AND email = $2`)).
			WithArgs(sqlmock.AnyArg(), driver.Value("user@example.com")).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery(regexp.QuoteMeta(
			`INSERT INTO "otp_requests" ("created_at","updated_at","email","otp","user_id","expires_at","is_active","is_verified") VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING "id"`)).
			WithArgs(
				sqlmock.AnyArg(),
				sqlmock.AnyArg(),
				driver.Value("user@example.com"),
				driver.Value("654321"),
				nil,
				sqlmock.AnyArg(),
				driver.Value(true),
				driver.Value(false),
			).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(2))
		mock.ExpectCommit()

		got, err := repo.Create(CreateOTPRequestInput{
			Email: strPtr("user@example.com"),
			OTP:   "654321",
		})

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, uint(2), got.ID)
		assert.Equal(t, "654321", got.OTP)
		assert.True(t, got.IsActive)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("deactivates existing active requests for user_id then creates new", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "otp_requests" SET "is_active"=$1,"updated_at"=$2 WHERE is_active = $3 AND user_id = $4`)).
			WithArgs(driver.Value(false), sqlmock.AnyArg(), driver.Value(true), driver.Value(uint(10))).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT count(*) FROM "otp_requests" WHERE created_at > $1 AND user_id = $2`)).
			WithArgs(sqlmock.AnyArg(), driver.Value(uint(10))).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery(regexp.QuoteMeta(
			`INSERT INTO "otp_requests" ("created_at","updated_at","email","otp","user_id","expires_at","is_active","is_verified") VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING "id"`)).
			WithArgs(
				sqlmock.AnyArg(),
				sqlmock.AnyArg(),
				nil,
				driver.Value("123456"),
				driver.Value(uint(10)),
				sqlmock.AnyArg(),
				driver.Value(true),
				driver.Value(false),
			).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(3))
		mock.ExpectCommit()

		got, err := repo.Create(CreateOTPRequestInput{
			UserID: uintPtr(10),
			OTP:    "123456",
		})

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, uint(3), got.ID)
		assert.Equal(t, "123456", got.OTP)
		assert.Equal(t, uintPtr(10), got.UserID)
		assert.True(t, got.IsActive)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when a recent request already exists", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "otp_requests" SET "is_active"=$1,"updated_at"=$2 WHERE is_active = $3 AND email = $4`)).
			WithArgs(driver.Value(false), sqlmock.AnyArg(), driver.Value(true), driver.Value("user@example.com")).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT count(*) FROM "otp_requests" WHERE created_at > $1 AND email = $2`)).
			WithArgs(sqlmock.AnyArg(), driver.Value("user@example.com")).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectRollback()

		got, err := repo.Create(CreateOTPRequestInput{
			Email: strPtr("user@example.com"),
			OTP:   "654321",
		})

		require.Error(t, err)
		require.Nil(t, got)
		assert.ErrorIs(t, err, httpx.ErrOTPRequestAlreadyMade)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("rolls back when deactivation update fails", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "otp_requests" SET "is_active"=$1,"updated_at"=$2 WHERE is_active = $3 AND email = $4`)).
			WithArgs(driver.Value(false), sqlmock.AnyArg(), driver.Value(true), driver.Value("user@example.com")).
			WillReturnError(assert.AnError)
		mock.ExpectRollback()

		got, err := repo.Create(CreateOTPRequestInput{
			Email: strPtr("user@example.com"),
			OTP:   "123456",
		})

		require.Error(t, err)
		require.Nil(t, got)
		assert.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("rolls back when insert fails", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "otp_requests" SET "is_active"=$1,"updated_at"=$2 WHERE is_active = $3 AND user_id = $4`)).
			WithArgs(driver.Value(false), sqlmock.AnyArg(), driver.Value(true), driver.Value(uint(7))).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT count(*) FROM "otp_requests" WHERE created_at > $1 AND user_id = $2`)).
			WithArgs(sqlmock.AnyArg(), driver.Value(uint(7))).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery(regexp.QuoteMeta(
			`INSERT INTO "otp_requests" ("created_at","updated_at","email","otp","user_id","expires_at","is_active","is_verified") VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING "id"`)).
			WithArgs(
				sqlmock.AnyArg(),
				sqlmock.AnyArg(),
				nil,
				driver.Value("999999"),
				driver.Value(uint(7)),
				sqlmock.AnyArg(),
				driver.Value(true),
				driver.Value(false),
			).
			WillReturnError(assert.AnError)
		mock.ExpectRollback()

		got, err := repo.Create(CreateOTPRequestInput{
			UserID: uintPtr(7),
			OTP:    "999999",
		})

		require.Error(t, err)
		require.Nil(t, got)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepo_MarkAsVerified_Unit(t *testing.T) {
	t.Run("sets is_verified and deactivates the request", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "otp_requests" SET "updated_at"=$1,"is_active"=$2,"is_verified"=$3 WHERE id = $4`)).
			WithArgs(sqlmock.AnyArg(), driver.Value(false), driver.Value(true), driver.Value(uint(3))).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := repo.MarkAsVerified(3)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error on update failure", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "otp_requests" SET "updated_at"=$1,"is_active"=$2,"is_verified"=$3 WHERE id = $4`)).
			WithArgs(sqlmock.AnyArg(), driver.Value(false), driver.Value(true), driver.Value(uint(3))).
			WillReturnError(assert.AnError)
		mock.ExpectRollback()

		err := repo.MarkAsVerified(3)

		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRepo_FindVerifiedByEmail_Unit(t *testing.T) {
	t.Run("returns verified request when found", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		cols := []string{"id", "email", "otp", "user_id", "expires_at", "is_active", "is_verified"}
		mock.ExpectQuery(`(?s)SELECT .* FROM "otp_requests" WHERE .* is_verified = .* is_active = .* ORDER BY created_at DESC.* LIMIT \$4`).
			WithArgs("user@example.com", true, false, 1).
			WillReturnRows(
				sqlmock.NewRows(cols).
					AddRow(1, "user@example.com", "123456", nil, time.Now().Add(time.Hour), false, true),
			)

		got, err := repo.FindVerifiedByEmail("USER@example.com")

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, uint(1), got.ID)
		assert.True(t, got.IsVerified)
		assert.False(t, got.IsActive)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when not found", func(t *testing.T) {
		gdb, mock := testutil.NewMockDB(t)
		repo := NewRepo(gdb)

		mock.ExpectQuery(`(?s)SELECT .* FROM "otp_requests" WHERE .* is_verified = .* is_active = .* ORDER BY created_at DESC.* LIMIT \$4`).
			WithArgs("missing@example.com", true, false, 1).
			WillReturnError(gorm.ErrRecordNotFound)

		got, err := repo.FindVerifiedByEmail("missing@example.com")

		require.Error(t, err)
		require.Nil(t, got)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
