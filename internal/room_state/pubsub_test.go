package roomstate

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"sound-stage-backend/internal/ws"
)

type mockBroadcaster struct{ mock.Mock }

func (m *mockBroadcaster) BroadcastToRoom(roomID uint, event ws.EventName, payload any) {
	m.Called(roomID, event, payload)
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newPubSubClient(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return mr, rdb
}

func TestRedisPublisher_Publish(t *testing.T) {
	t.Run("success: publishes encoded message to the channel", func(t *testing.T) {
		_, rdb := newPubSubClient(t)
		ctx := context.Background()
		sub := rdb.Subscribe(ctx, DefaultChannel)
		t.Cleanup(func() { _ = sub.Close() })
		_, err := sub.Receive(ctx)
		require.NoError(t, err)

		pub := NewPublisher(rdb, DefaultChannel, testLogger())
		payload := ParticipantState{UserID: 42, IsMuted: true}
		require.NoError(t, pub.Publish(ctx, 4, 42, ws.EventSetMuted, payload))

		msg, err := sub.ReceiveMessage(ctx)
		require.NoError(t, err)

		var m Message
		require.NoError(t, json.Unmarshal([]byte(msg.Payload), &m))
		assert.Equal(t, uint(4), m.RoomID)
		assert.Equal(t, uint(42), m.UserID)
		assert.Equal(t, ws.EventSetMuted, m.Event)

		var decoded ParticipantState
		require.NoError(t, json.Unmarshal(m.Payload, &decoded))
		assert.Equal(t, payload, decoded)
	})

	t.Run("success: empty channel falls back to the default channel", func(t *testing.T) {
		_, rdb := newPubSubClient(t)
		ctx := context.Background()
		sub := rdb.Subscribe(ctx, DefaultChannel)
		t.Cleanup(func() { _ = sub.Close() })
		_, err := sub.Receive(ctx)
		require.NoError(t, err)

		pub := NewPublisher(rdb, "", testLogger())
		require.NoError(t, pub.Publish(ctx, 4, 42, ws.EventSetHandRaised, nil))

		msg, err := sub.ReceiveMessage(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, msg.Payload)
	})

	t.Run("failure: unmarshalable payload returns error and publishes nothing", func(t *testing.T) {
		_, rdb := newPubSubClient(t)
		ctx := context.Background()
		sub := rdb.Subscribe(ctx, DefaultChannel)
		t.Cleanup(func() { _ = sub.Close() })
		_, err := sub.Receive(ctx)
		require.NoError(t, err)

		pub := NewPublisher(rdb, DefaultChannel, testLogger())

		require.Error(t, pub.Publish(ctx, 4, 42, ws.EventSetMuted, func() {}))

		msgCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
		defer cancel()
		_, err = sub.ReceiveMessage(msgCtx)
		require.Error(t, err)
	})
}

func TestSubscriber_handle(t *testing.T) {
	t.Run("success: decodes message and broadcasts to room", func(t *testing.T) {
		broadcaster := new(mockBroadcaster)
		s := NewSubscriber(nil, DefaultChannel, broadcaster, testLogger())
		raw, err := json.Marshal(Message{
			RoomID:  4,
			UserID:  42,
			Event:   ws.EventSetMuted,
			Payload: json.RawMessage(`{"userId":42,"isMuted":true}`),
		})
		require.NoError(t, err)

		broadcaster.On("BroadcastToRoom", uint(4), ws.EventSetMuted, mock.Anything).Return()

		require.NoError(t, s.handle(&redis.Message{Payload: string(raw)}))

		broadcaster.AssertExpectations(t)
	})

	t.Run("failure: malformed payload returns error, no broadcast", func(t *testing.T) {
		broadcaster := new(mockBroadcaster)
		s := NewSubscriber(nil, DefaultChannel, broadcaster, testLogger())

		err := s.handle(&redis.Message{Payload: "{not json"})

		require.Error(t, err)
		broadcaster.AssertNotCalled(t, "BroadcastToRoom", mock.Anything, mock.Anything, mock.Anything)
	})
}

func TestSubscriber_Run(t *testing.T) {
	t.Run("success: broadcasts published messages until context is cancelled", func(t *testing.T) {
		_, rdb := newPubSubClient(t)
		broadcaster := new(mockBroadcaster)
		received := make(chan struct{})
		broadcaster.On("BroadcastToRoom", uint(4), ws.EventSetMuted, mock.Anything).
			Run(func(mock.Arguments) { close(received) }).Return()

		s := NewSubscriber(rdb, DefaultChannel, broadcaster, testLogger())
		ctx, cancel := context.WithCancel(context.Background())
		runErr := make(chan error, 1)
		go func() { runErr <- s.Run(ctx) }()

		// wait until the subscription is active before publishing
		require.Eventually(t, func() bool {
			return rdb.PubSubNumSub(context.Background(), DefaultChannel).Val()[DefaultChannel] > 0
		}, 2*time.Second, 10*time.Millisecond)

		pub := NewPublisher(rdb, DefaultChannel, testLogger())
		require.NoError(t, pub.Publish(context.Background(), 4, 42, ws.EventSetMuted, nil))

		select {
		case <-received:
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for broadcast")
		}

		cancel()
		require.ErrorIs(t, <-runErr, context.Canceled)
		broadcaster.AssertExpectations(t)
	})

	t.Run("failure: malformed message is logged, subscriber keeps running", func(t *testing.T) {
		_, rdb := newPubSubClient(t)
		broadcaster := new(mockBroadcaster)
		received := make(chan struct{})
		broadcaster.On("BroadcastToRoom", uint(4), ws.EventSetMuted, mock.Anything).
			Run(func(mock.Arguments) { close(received) }).Return()

		s := NewSubscriber(rdb, DefaultChannel, broadcaster, testLogger())
		ctx, cancel := context.WithCancel(context.Background())
		runErr := make(chan error, 1)
		go func() { runErr <- s.Run(ctx) }()

		require.Eventually(t, func() bool {
			return rdb.PubSubNumSub(context.Background(), DefaultChannel).Val()[DefaultChannel] > 0
		}, 2*time.Second, 10*time.Millisecond)

		require.NoError(t, rdb.Publish(context.Background(), DefaultChannel, "{not json").Err())
		pub := NewPublisher(rdb, DefaultChannel, testLogger())
		require.NoError(t, pub.Publish(context.Background(), 4, 42, ws.EventSetMuted, nil))

		select {
		case <-received:
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for broadcast after malformed message")
		}

		cancel()
		require.ErrorIs(t, <-runErr, context.Canceled)
		broadcaster.AssertExpectations(t)
	})
}

func TestSubscriber_Shutdown(t *testing.T) {
	t.Run("success: no-op when never run", func(t *testing.T) {
		s := NewSubscriber(nil, DefaultChannel, new(mockBroadcaster), testLogger())
		require.NoError(t, s.Shutdown(context.Background()))
	})

	t.Run("success: closes an active subscription", func(t *testing.T) {
		_, rdb := newPubSubClient(t)
		s := NewSubscriber(rdb, DefaultChannel, new(mockBroadcaster), testLogger())
		runErr := make(chan error, 1)
		go func() { runErr <- s.Run(context.Background()) }()

		require.Eventually(t, func() bool {
			return rdb.PubSubNumSub(context.Background(), DefaultChannel).Val()[DefaultChannel] > 0
		}, 2*time.Second, 10*time.Millisecond)

		require.NoError(t, s.Shutdown(context.Background()))
		require.NoError(t, <-runErr)
	})
}
