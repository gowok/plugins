package redis

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/ngamux/middleware/idempotency"
	goredis "github.com/redis/go-redis/v9"
)

const (
	idempotencyKeyPrefix  = "gowok:idempotency:"
	idempotencyInProgress = "in-progress"
	defaultIdempotencyTTL = 24 * time.Hour
)

type StoreIdempotency struct {
	Name string
	TTL  time.Duration
}

var _ idempotency.Store = StoreIdempotency{}

func (s StoreIdempotency) Begin(key string) (*idempotency.StoredResponse, idempotency.State) {
	client, ok := s.client()
	if !ok {
		return nil, idempotency.StateMissing
	}

	ctx, cancel := context.WithTimeout(context.Background(), healthCheckTimeout)
	defer cancel()

	k := idempotencyKeyPrefix + key
	created, err := client.SetNX(ctx, k, idempotencyInProgress, 0).Result()
	if err != nil {
		slog.Warn("failed to start idempotency entry", "plugin", plugin, "error", err)
		return nil, idempotency.StateMissing
	}
	if created {
		return nil, idempotency.StateMissing
	}

	data, err := client.Get(ctx, k).Bytes()
	if errors.Is(err, goredis.Nil) {
		return nil, idempotency.StateMissing
	}
	if err != nil {
		slog.Warn("failed to read idempotency entry", "plugin", plugin, "error", err)
		return nil, idempotency.StateMissing
	}
	if string(data) == idempotencyInProgress {
		return nil, idempotency.StateInProgress
	}

	var response idempotency.StoredResponse
	if err := json.Unmarshal(data, &response); err != nil {
		slog.Warn("failed to decode idempotency entry", "plugin", plugin, "error", err)
		return nil, idempotency.StateMissing
	}

	return &response, idempotency.StateFound
}

func (s StoreIdempotency) Complete(key string, response *idempotency.StoredResponse) {
	if response == nil {
		s.Abort(key)
		return
	}

	client, ok := s.client()
	if !ok {
		return
	}

	data, err := json.Marshal(response)
	if err != nil {
		slog.Warn("failed to encode idempotency response", "plugin", plugin, "error", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), healthCheckTimeout)
	defer cancel()

	if err := client.Set(ctx, idempotencyKeyPrefix+key, data, s.ttl()).Err(); err != nil {
		slog.Warn("failed to store idempotency response", "plugin", plugin, "error", err)
	}
}

func (s StoreIdempotency) Abort(key string) {
	client, ok := s.client()
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), healthCheckTimeout)
	defer cancel()

	if err := client.Del(ctx, idempotencyKeyPrefix+key).Err(); err != nil {
		slog.Warn("failed to abort idempotency entry", "plugin", plugin, "error", err)
	}
}

func (s StoreIdempotency) ttl() time.Duration {
	if s.TTL > 0 {
		return s.TTL
	}
	return defaultIdempotencyTTL
}

func (s StoreIdempotency) client() (*goredis.Client, bool) {
	name := s.Name
	if name == "" {
		name = "default"
	}

	client, ok := Client(name).Get()
	return client, ok && client != nil
}
