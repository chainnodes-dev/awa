package events

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/redis/go-redis/v9"
)

const redisChannel = "asm:events"

// Bus is the event distribution interface.
type Bus interface {
	Publish(ctx context.Context, event Event) error
	Subscribe(ctx context.Context) (<-chan Event, func())
}

// LocalBus is an in-process bus using channels — used in direct/dev mode.
type LocalBus struct {
	mu          sync.RWMutex
	subscribers map[int]chan Event
	next        int
}

func NewLocalBus() *LocalBus {
	return &LocalBus{subscribers: make(map[int]chan Event)}
}

func (b *LocalBus) Publish(_ context.Context, event Event) error {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.subscribers {
		select {
		case ch <- event:
		default:
			// drop if subscriber is slow
		}
	}
	return nil
}

func (b *LocalBus) Subscribe(_ context.Context) (<-chan Event, func()) {
	ch := make(chan Event, 64)
	b.mu.Lock()
	id := b.next
	b.next++
	b.subscribers[id] = ch
	b.mu.Unlock()

	cancel := func() {
		b.mu.Lock()
		delete(b.subscribers, id)
		b.mu.Unlock()
		close(ch)
	}
	return ch, cancel
}

// RedisBus publishes events through Redis Pub/Sub for multi-process deployments.
type RedisBus struct {
	client *redis.Client
}

func NewRedisBus(redisURL string) (*RedisBus, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis URL: %w", err)
	}
	return &RedisBus{client: redis.NewClient(opts)}, nil
}

func (b *RedisBus) Publish(ctx context.Context, event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return b.client.Publish(ctx, redisChannel, data).Err()
}

func (b *RedisBus) Subscribe(ctx context.Context) (<-chan Event, func()) {
	sub := b.client.Subscribe(ctx, redisChannel)
	ch := make(chan Event, 64)

	go func() {
		defer close(ch)
		redisCh := sub.Channel()
		for msg := range redisCh {
			var event Event
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				continue
			}
			select {
			case ch <- event:
			case <-ctx.Done():
				return
			}
		}
	}()

	cancel := func() {
		_ = sub.Close()
	}
	return ch, cancel
}

// Ping checks that the Redis connection is healthy.
func (b *RedisBus) Ping(ctx context.Context) error {
	return b.client.Ping(ctx).Err()
}
