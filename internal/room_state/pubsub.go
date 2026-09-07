package roomstate

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sound-stage-backend/internal/ws"
	"sync"

	"github.com/redis/go-redis/v9"
)

const DefaultChannel = "room_state:updates"

type Broadcaster interface {
	BroadcastToRoom(roomID uint, event ws.EventName, payload any)
}

type Publisher interface {
	Publish(ctx context.Context, roomID, userID uint, event ws.EventName, payload any) error
}

type Message struct {
	RoomID  uint            `json:"room_id"`
	UserID  uint            `json:"user_id"`
	Event   ws.EventName    `json:"event"`
	Payload json.RawMessage `json:"payload"`
}

type redisPublisher struct {
	rdb     *redis.Client
	channel string
	logger  *slog.Logger
}

func NewPublisher(rdb *redis.Client, channel string, logger *slog.Logger) Publisher {
	if channel == "" {
		channel = DefaultChannel
	}
	return &redisPublisher{rdb: rdb, channel: channel, logger: logger}
}

func (p *redisPublisher) Publish(ctx context.Context, roomID, userID uint, event ws.EventName, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("roomstate: marshal payload: %w", err)
	}

	msg := Message{RoomID: roomID, UserID: userID, Event: event, Payload: raw}
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("roomstate: marshal message: %w", err)
	}

	if err := p.rdb.Publish(ctx, p.channel, data).Err(); err != nil {
		return fmt.Errorf("roomstate: publish: %w", err)
	}
	return nil
}

type Subscriber struct {
	rdb         *redis.Client
	channel     string
	broadcaster Broadcaster
	logger      *slog.Logger
	mu          sync.Mutex
	pubsub      *redis.PubSub
}

func NewSubscriber(rdb *redis.Client, channel string, broadcaster Broadcaster, logger *slog.Logger) *Subscriber {
	if channel == "" {
		channel = DefaultChannel
	}
	return &Subscriber{rdb: rdb, channel: channel, broadcaster: broadcaster, logger: logger}
}

func (s *Subscriber) Run(ctx context.Context) error {
	pubsub := s.rdb.Subscribe(ctx, s.channel)
	s.mu.Lock()
	s.pubsub = pubsub
	s.mu.Unlock()
	ch := pubsub.Channel()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-ch:
			if !ok {
				return nil
			}
			if err := s.handle(msg); err != nil {
				s.logger.Error("roomstate: handle message", slog.Any("error", err))
			}
		}
	}
}

func (s *Subscriber) Shutdown(context.Context) error {
	s.mu.Lock()
	pubsub := s.pubsub
	s.mu.Unlock()
	if pubsub == nil {
		return nil
	}
	return pubsub.Close()
}

func (s *Subscriber) handle(msg *redis.Message) error {
	var m Message
	if err := json.Unmarshal([]byte(msg.Payload), &m); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	s.broadcaster.BroadcastToRoom(m.RoomID, m.Event, m.Payload)
	return nil
}
