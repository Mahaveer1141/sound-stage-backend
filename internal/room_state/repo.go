package roomstate

import (
	"context"
	"fmt"
	"sound-stage-backend/internal/pkg/listopts"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	fieldIsMuted      = "is_muted"
	fieldIsHandRaised = "is_hand_raised"
)

const DefaultIsHandRaised = false
const DefaultIsMuted = true

type RedisRepo struct {
	rdb *redis.Client
}

func NewRedisRepo(rdb *redis.Client) *RedisRepo {
	return &RedisRepo{rdb: rdb}
}

func (r *RedisRepo) SetMuted(ctx context.Context, roomID, userID uint, isMuted bool) (*ParticipantState, error) {
	key := participantKey(roomID, userID)

	cmds, err := r.rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.HSet(ctx, key, fieldIsMuted, isMuted)
		pipe.HSetNX(ctx, key, fieldIsHandRaised, DefaultIsHandRaised)
		pipe.HGetAll(ctx, key)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("roomstate: set muted: %w", err)
	}

	vals, err := cmds[2].(*redis.MapStringStringCmd).Result()
	if err != nil {
		return nil, fmt.Errorf("roomstate: set muted: %w", err)
	}
	return parseParticipantState(userID, vals), nil
}

func (r *RedisRepo) SetHandRaised(ctx context.Context, roomID, userID uint, isHandRaised bool) (*ParticipantState, error) {
	key := participantKey(roomID, userID)

	cmds, err := r.rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.HSet(ctx, key, fieldIsHandRaised, isHandRaised)
		pipe.HSetNX(ctx, key, fieldIsMuted, DefaultIsMuted)
		if isHandRaised {
			pipe.ZAdd(ctx, raisedHandsKey(roomID), redis.Z{
				Score:  float64(time.Now().UnixMilli()),
				Member: userID,
			})
		} else {
			pipe.ZRem(ctx, raisedHandsKey(roomID), userID)
		}
		pipe.HGetAll(ctx, key)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("roomstate: set hand raised: %w", err)
	}

	vals, err := cmds[3].(*redis.MapStringStringCmd).Result()
	if err != nil {
		return nil, fmt.Errorf("roomstate: set hand raised: %w", err)
	}
	return parseParticipantState(userID, vals), nil
}

func (r *RedisRepo) GetParticipantStates(ctx context.Context, roomID uint, userIDs []uint) (map[uint]*ParticipantState, error) {
	states := make(map[uint]*ParticipantState, len(userIDs))
	if len(userIDs) == 0 {
		return states, nil
	}

	cmds, err := r.rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for _, userID := range userIDs {
			pipe.HGetAll(ctx, participantKey(roomID, userID))
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("roomstate: get participant states: %w", err)
	}

	for i, cmd := range cmds {
		vals, err := cmd.(*redis.MapStringStringCmd).Result()
		if err != nil {
			return nil, fmt.Errorf("roomstate: get participant states: %w", err)
		}
		states[userIDs[i]] = parseParticipantState(userIDs[i], vals)
	}
	return states, nil
}

func (r *RedisRepo) DeleteParticipantState(ctx context.Context, roomID, userID uint) error {
	_, err := r.rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Del(ctx, participantKey(roomID, userID))
		pipe.ZRem(ctx, raisedHandsKey(roomID), userID)
		return nil
	})
	if err != nil {
		return fmt.Errorf("roomstate: delete participant state: %w", err)
	}
	return nil
}

func (r *RedisRepo) DeleteRoomState(ctx context.Context, roomID uint) error {
	pattern := fmt.Sprintf("room:%d:user:*", roomID)
	keys := []string{raisedHandsKey(roomID)}

	var cursor uint64
	for {
		found, next, err := r.rdb.Scan(ctx, cursor, pattern, 500).Result()
		if err != nil {
			return fmt.Errorf("roomstate: delete room state: %w", err)
		}
		keys = append(keys, found...)
		cursor = next
		if cursor == 0 {
			break
		}
	}

	if err := r.rdb.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("roomstate: delete room state: %w", err)
	}
	return nil
}

func (r *RedisRepo) GetRaisedHands(ctx context.Context, roomID uint, p listopts.Pagination) ([]uint, error) {
	start := int64(p.Offset())
	end := start + int64(p.PageSize) - 1
	members, err := r.rdb.ZRange(ctx, raisedHandsKey(roomID), start, end).Result()
	if err != nil {
		return nil, fmt.Errorf("roomstate: get raised hands: %w", err)
	}

	ids := make([]uint, 0, len(members))
	for _, m := range members {
		n, err := strconv.ParseUint(m, 10, 0)
		if err != nil {
			continue
		}
		ids = append(ids, uint(n))
	}
	return ids, nil
}

func (r *RedisRepo) CountRaisedHands(ctx context.Context, roomID uint) (int64, error) {
	count, err := r.rdb.ZCard(ctx, raisedHandsKey(roomID)).Result()
	if err != nil {
		return 0, fmt.Errorf("roomstate: count raised hands: %w", err)
	}
	return count, nil
}

func parseParticipantState(userID uint, vals map[string]string) *ParticipantState {
	state := &ParticipantState{
		UserID:       userID,
		IsMuted:      DefaultIsMuted,
		IsHandRaised: DefaultIsHandRaised,
	}
	if v, ok := vals[fieldIsMuted]; ok {
		state.IsMuted, _ = strconv.ParseBool(v)
	}
	if v, ok := vals[fieldIsHandRaised]; ok {
		state.IsHandRaised, _ = strconv.ParseBool(v)
	}
	return state
}

func participantKey(roomID, userID uint) string {
	return fmt.Sprintf("room:%d:user:%d", roomID, userID)
}

func raisedHandsKey(roomID uint) string {
	return fmt.Sprintf("raised_hands:room:%d", roomID)
}
