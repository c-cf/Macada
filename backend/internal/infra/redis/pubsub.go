package redis

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cchu-code/managed-agents/internal/domain"
	"github.com/redis/go-redis/v9"
)

type EventBus struct {
	client *redis.Client
}

func NewEventBus(client *redis.Client) *EventBus {
	return &EventBus{client: client}
}

func channelName(sessionID string) string {
	return fmt.Sprintf("session:%s:events", sessionID)
}

func (b *EventBus) Publish(ctx context.Context, sessionID string, event *domain.Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	return b.client.Publish(ctx, channelName(sessionID), data).Err()
}

func (b *EventBus) Subscribe(ctx context.Context, sessionID string) (<-chan *domain.Event, func(), error) {
	sub := b.client.Subscribe(ctx, channelName(sessionID))

	// Wait for subscription confirmation
	if _, err := sub.Receive(ctx); err != nil {
		_ = sub.Close()
		return nil, nil, fmt.Errorf("subscribe: %w", err)
	}

	ch := make(chan *domain.Event, 64)
	subCtx, cancel := context.WithCancel(ctx)

	go func() {
		defer close(ch)
		msgCh := sub.Channel()
		for {
			select {
			case msg, ok := <-msgCh:
				if !ok {
					return
				}
				var evt domain.Event
				if err := json.Unmarshal([]byte(msg.Payload), &evt); err != nil {
					continue
				}
				select {
				case ch <- &evt:
				case <-subCtx.Done():
					return
				}
			case <-subCtx.Done():
				_ = sub.Close()
				return
			}
		}
	}()

	cleanup := func() {
		cancel()
	}

	return ch, cleanup, nil
}

func NewRedisClient(redisURL string) (*redis.Client, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis URL: %w", err)
	}
	client := redis.NewClient(opts)
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return client, nil
}
