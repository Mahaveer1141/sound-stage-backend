package user

import (
	"testing"

	"sound-stage-backend/internal/pkg/testutil"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestRepo_Create_Integration(t *testing.T) {
	t.Run("persists a new user", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &User{})
		repo := NewRepo(db)

		got, err := repo.Create(&CreateUserParams{
			Email:     "user@example.com",
			FirstName: "First",
			LastName:  "Last",
		})

		require.NoError(t, err)
		require.NotNil(t, got)
		require.NotZero(t, got.ID)
		require.Equal(t, "user@example.com", got.Email)
		require.Equal(t, "First", got.FirstName)
		require.NotNil(t, got.LastName)
		require.Equal(t, "Last", *got.LastName)

		var fetched User
		require.NoError(t, db.First(&fetched, got.ID).Error)
		require.Equal(t, got.Email, fetched.Email)
	})

	t.Run("returns error for duplicate email", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &User{})
		repo := NewRepo(db)

		_, err := repo.Create(&CreateUserParams{
			Email:     "user@example.com",
			FirstName: "First",
		})
		require.NoError(t, err)

		got, err := repo.Create(&CreateUserParams{
			Email:     "user@example.com",
			FirstName: "Other",
		})

		require.Error(t, err)
		require.Nil(t, got)
	})

	t.Run("returns error for duplicate email with different casing", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &User{})
		repo := NewRepo(db)

		_, err := repo.Create(&CreateUserParams{
			Email:     "USER@example.com",
			FirstName: "First",
		})
		require.NoError(t, err)

		got, err := repo.Create(&CreateUserParams{
			Email:     "user@example.com",
			FirstName: "Other",
		})

		require.Error(t, err)
		require.Nil(t, got)
	})
}

func TestRepo_FindByEmail_Integration(t *testing.T) {
	t.Run("finds user by email case-insensitively", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &User{})
		repo := NewRepo(db)

		_, err := repo.Create(&CreateUserParams{
			Email:     "user@example.com",
			FirstName: "First",
		})
		require.NoError(t, err)

		got, err := repo.FindByEmail("USER@Example.com")

		require.NoError(t, err)
		require.NotNil(t, got)
		require.Equal(t, "user@example.com", got.Email)
	})

	t.Run("returns nil when not found", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &User{})
		repo := NewRepo(db)

		got, err := repo.FindByEmail("missing@example.com")

		require.NoError(t, err)
		require.Nil(t, got)
	})
}

func TestRepo_FindByID_Integration(t *testing.T) {
	t.Run("finds user by id", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &User{})
		repo := NewRepo(db)

		created, err := repo.Create(&CreateUserParams{
			Email:     "user@example.com",
			FirstName: "First",
		})
		require.NoError(t, err)

		got, err := repo.FindByID(created.ID)

		require.NoError(t, err)
		require.NotNil(t, got)
		require.Equal(t, created.ID, got.ID)
		require.Equal(t, "user@example.com", got.Email)
	})

	t.Run("returns nil when not found", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &User{})
		repo := NewRepo(db)

		got, err := repo.FindByID(999)

		require.NoError(t, err)
		require.Nil(t, got)
	})
}

func TestRepo_UpdateLastLoginAt_Integration(t *testing.T) {
	t.Run("sets last login timestamp", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &User{})
		repo := NewRepo(db)

		created, err := repo.Create(&CreateUserParams{
			Email:     "user@example.com",
			FirstName: "First",
		})
		require.NoError(t, err)

		require.NoError(t, repo.UpdateLastLoginAt(created.ID))

		var fetched User
		require.NoError(t, db.First(&fetched, created.ID).Error)
		require.NotNil(t, fetched.LastLoginAt)
		require.False(t, fetched.LastLoginAt.IsZero())
	})
}

func TestRepo_Update_Integration(t *testing.T) {
	t.Run("updates first and last name", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &User{})
		repo := NewRepo(db)

		created, err := repo.Create(&CreateUserParams{
			Email:     "user@example.com",
			FirstName: "First",
			LastName:  "Last",
		})
		require.NoError(t, err)

		got, err := repo.Update(created.ID, &UpdateUserParams{
			FirstName: "Updated",
			LastName:  "New",
		})

		require.NoError(t, err)
		require.NotNil(t, got)
		require.Equal(t, "Updated", got.FirstName)
		require.NotNil(t, got.LastName)
		require.Equal(t, "New", *got.LastName)

		var fetched User
		require.NoError(t, db.First(&fetched, created.ID).Error)
		require.Equal(t, "Updated", fetched.FirstName)
	})

	t.Run("returns error for non-existent user", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &User{})
		repo := NewRepo(db)

		got, err := repo.Update(999, &UpdateUserParams{FirstName: "Updated"})

		require.Error(t, err)
		require.Nil(t, got)
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})
}
