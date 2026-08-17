package apitoken

import (
	"sound-stage-backend/internal/pkg/testutil"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func seedAPITokens(t *testing.T, db *gorm.DB, tokens ...APIToken) {
	t.Helper()
	for i := range tokens {
		require.NoError(t, db.Create(&tokens[i]).Error)
	}
}

func TestRepo_FindByToken_Integration(t *testing.T) {
	t.Run("finds an active seeded token", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &APIToken{})
		seedAPITokens(t, db,
			APIToken{Token: "active-tok", Type: string(AccessToken), UserID: 1, IsActive: true},
			APIToken{Token: "inactive-tok", Type: string(AccessToken), UserID: 1, IsActive: false},
		)
		repo := NewRepo(db)

		got, err := repo.FindByToken("active-tok")

		require.NoError(t, err)
		require.NotNil(t, got)
		require.Equal(t, "active-tok", got.Token)
		require.NotZero(t, got.ID)
	})

	t.Run("does not find an inactive token", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &APIToken{})
		seedAPITokens(t, db,
			APIToken{Token: "inactive-tok", Type: string(AccessToken), UserID: 1, IsActive: false},
		)
		repo := NewRepo(db)

		got, err := repo.FindByToken("inactive-tok")

		require.Error(t, err)
		require.Nil(t, got)
	})
}

func TestRepo_CreateToken_Integration(t *testing.T) {
	t.Run("persists a new active token", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &APIToken{})
		repo := NewRepo(db)

		got, err := repo.CreateToken(CreateAPITokenInput{
			Token:  "new-tok",
			Type:   AccessToken,
			UserID: 3,
		})

		require.NoError(t, err)
		require.NotNil(t, got)
		require.NotZero(t, got.ID)
		require.True(t, got.IsActive)

		var fetched APIToken
		require.NoError(t, db.First(&fetched, got.ID).Error)
		require.Equal(t, "new-tok", fetched.Token)
	})
}

func TestRepo_Deactivate_Integration(t *testing.T) {
	t.Run("deactivates only the target user's active tokens", func(t *testing.T) {
		db := testutil.NewIntegrationDB(t, &APIToken{})
		seedAPITokens(t, db,
			APIToken{Token: "tok-1", Type: string(AccessToken), UserID: 5, IsActive: true},
			APIToken{Token: "tok-2", Type: string(AccessToken), UserID: 5, IsActive: true},
			APIToken{Token: "tok-3", Type: string(AccessToken), UserID: 9, IsActive: true},
		)
		repo := NewRepo(db)

		err := repo.Deactivate(5)
		require.NoError(t, err)

		var userFive []APIToken
		require.NoError(t, db.Where("user_id = ?", 5).Find(&userFive).Error)
		for _, tok := range userFive {
			require.False(t, tok.IsActive)
		}

		var userNine APIToken
		require.NoError(t, db.Where("user_id = ?", 9).First(&userNine).Error)
		require.True(t, userNine.IsActive)
	})
}
