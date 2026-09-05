package roomstate

import (
	"context"
	"sound-stage-backend/internal/pkg/listopts"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newRedisRepoHarness(t *testing.T) (*miniredis.Miniredis, *redis.Client, *RedisRepo) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return mr, rdb, NewRedisRepo(rdb)
}

func zsetMembers(t *testing.T, rdb *redis.Client, key string) []string {
	t.Helper()
	members, err := rdb.ZRange(context.Background(), key, 0, -1).Result()
	require.NoError(t, err)
	return members
}

func TestRedisRepo_SetMuted_Integration(t *testing.T) {
	t.Run("persists mute state and applies hand-raised default", func(t *testing.T) {
		_, _, repo := newRedisRepoHarness(t)
		ctx := context.Background()

		got, err := repo.SetMuted(ctx, 4, 42, false)

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, uint(42), got.UserID)
		assert.False(t, got.IsMuted)
		assert.Equal(t, DefaultIsHandRaised, got.IsHandRaised)
	})

	t.Run("preserves existing hand-raised value", func(t *testing.T) {
		_, rdb, repo := newRedisRepoHarness(t)
		ctx := context.Background()
		require.NoError(t, rdb.HSet(ctx, participantKey(4, 42), fieldIsHandRaised, true).Err())

		got, err := repo.SetMuted(ctx, 4, 42, true)

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.True(t, got.IsMuted)
		assert.True(t, got.IsHandRaised)
	})
}

func TestRedisRepo_SetHandRaised_Integration(t *testing.T) {
	t.Run("raised hand adds user to raised set and applies muted default", func(t *testing.T) {
		_, rdb, repo := newRedisRepoHarness(t)
		ctx := context.Background()

		got, err := repo.SetHandRaised(ctx, 4, 42, true)

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.True(t, got.IsHandRaised)
		assert.Equal(t, DefaultIsMuted, got.IsMuted)
		assert.ElementsMatch(t, []string{"42"}, zsetMembers(t, rdb, raisedHandsKey(4)))
	})

	t.Run("lowered hand removes user from raised set", func(t *testing.T) {
		_, rdb, repo := newRedisRepoHarness(t)
		ctx := context.Background()
		require.NoError(t, rdb.ZAdd(ctx, raisedHandsKey(4), redis.Z{Score: 1, Member: 42}).Err())

		got, err := repo.SetHandRaised(ctx, 4, 42, false)

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.False(t, got.IsHandRaised)
		assert.Empty(t, zsetMembers(t, rdb, raisedHandsKey(4)))
	})
}

func TestRedisRepo_DeleteParticipantState_Integration(t *testing.T) {
	t.Run("removes hash and raised-hands membership", func(t *testing.T) {
		_, rdb, repo := newRedisRepoHarness(t)
		ctx := context.Background()
		require.NoError(t, rdb.HSet(ctx, participantKey(4, 42), fieldIsMuted, true).Err())
		require.NoError(t, rdb.ZAdd(ctx, raisedHandsKey(4), redis.Z{Score: 1, Member: 42}).Err())

		err := repo.DeleteParticipantState(ctx, 4, 42)

		require.NoError(t, err)
		exists, err := rdb.Exists(ctx, participantKey(4, 42)).Result()
		require.NoError(t, err)
		assert.Zero(t, exists)
		assert.Empty(t, zsetMembers(t, rdb, raisedHandsKey(4)))
	})
}

func TestRedisRepo_GetRaisedHands_Integration(t *testing.T) {
	t.Run("returns empty list when no hands raised", func(t *testing.T) {
		_, _, repo := newRedisRepoHarness(t)

		got, err := repo.GetRaisedHands(context.Background(), 4, listopts.Pagination{Page: 1, PageSize: 10})

		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("returns member ids in ascending score order and skips non-numeric entries", func(t *testing.T) {
		_, rdb, repo := newRedisRepoHarness(t)
		ctx := context.Background()
		require.NoError(t, rdb.ZAdd(ctx, raisedHandsKey(4),
			redis.Z{Score: 2, Member: 9},
			redis.Z{Score: 1, Member: 7},
			redis.Z{Score: 3, Member: "not-a-number"},
		).Err())

		got, err := repo.GetRaisedHands(ctx, 4, listopts.Pagination{Page: 1, PageSize: 10})

		require.NoError(t, err)
		assert.Equal(t, []uint{7, 9}, got)
	})

	t.Run("paginates members by page and page size", func(t *testing.T) {
		_, rdb, repo := newRedisRepoHarness(t)
		ctx := context.Background()
		require.NoError(t, rdb.ZAdd(ctx, raisedHandsKey(4),
			redis.Z{Score: 1, Member: 3},
			redis.Z{Score: 2, Member: 5},
			redis.Z{Score: 3, Member: 7},
			redis.Z{Score: 4, Member: 9},
			redis.Z{Score: 5, Member: 11},
		).Err())

		got, err := repo.GetRaisedHands(ctx, 4, listopts.Pagination{Page: 2, PageSize: 2})

		require.NoError(t, err)
		assert.Equal(t, []uint{7, 9}, got)
	})
}

func TestRedisRepo_CountRaisedHands_Integration(t *testing.T) {
	t.Run("returns zero when no hands raised", func(t *testing.T) {
		_, _, repo := newRedisRepoHarness(t)

		got, err := repo.CountRaisedHands(context.Background(), 4)

		require.NoError(t, err)
		assert.Zero(t, got)
	})

	t.Run("returns total member count", func(t *testing.T) {
		_, rdb, repo := newRedisRepoHarness(t)
		ctx := context.Background()
		require.NoError(t, rdb.ZAdd(ctx, raisedHandsKey(4),
			redis.Z{Score: 1, Member: 7},
			redis.Z{Score: 2, Member: 9},
		).Err())

		got, err := repo.CountRaisedHands(ctx, 4)

		require.NoError(t, err)
		assert.Equal(t, int64(2), got)
	})
}

func TestRedisRepo_GetParticipantStates_Integration(t *testing.T) {
	t.Run("returns empty map for no user ids", func(t *testing.T) {
		_, _, repo := newRedisRepoHarness(t)

		got, err := repo.GetParticipantStates(context.Background(), 4, nil)

		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("returns per-user states, nil for unknown participants", func(t *testing.T) {
		_, rdb, repo := newRedisRepoHarness(t)
		ctx := context.Background()
		require.NoError(t, rdb.HSet(ctx, participantKey(4, 7), fieldIsMuted, true, fieldIsHandRaised, true).Err())

		got, err := repo.GetParticipantStates(ctx, 4, []uint{7, 99})

		require.NoError(t, err)
		require.Len(t, got, 2)
		require.NotNil(t, got[7])
		assert.True(t, got[7].IsMuted)
		assert.True(t, got[7].IsHandRaised)
		assert.Nil(t, got[99])
	})
}
