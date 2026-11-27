package memcached

import (
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/ngamux/middleware/idempotency"
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
	c, ok := s.client()
	if !ok {
		return nil, idempotency.StateMissing
	}

	k := idempotencyKeyPrefix + key
	err := c.Add(&memcache.Item{
		Key:   k,
		Value: []byte(idempotencyInProgress),
	})
	if err == nil {
		return nil, idempotency.StateMissing
	}
	if !errors.Is(err, memcache.ErrNotStored) {
		slog.Warn("failed to start idempotency entry", "plugin", plugin, "error", err)
		return nil, idempotency.StateMissing
	}

	item, err := c.Get(k)
	if errors.Is(err, memcache.ErrCacheMiss) {
		return nil, idempotency.StateMissing
	}
	if err != nil {
		slog.Warn("failed to read idempotency entry", "plugin", plugin, "error", err)
		return nil, idempotency.StateMissing
	}
	if string(item.Value) == idempotencyInProgress {
		return nil, idempotency.StateInProgress
	}

	var response idempotency.StoredResponse
	if err := json.Unmarshal(item.Value, &response); err != nil {
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

	c, ok := s.client()
	if !ok {
		return
	}

	data, err := json.Marshal(response)
	if err != nil {
		slog.Warn("failed to encode idempotency response", "plugin", plugin, "error", err)
		return
	}
	if err := c.Set(&memcache.Item{
		Key:        idempotencyKeyPrefix + key,
		Value:      data,
		Expiration: expirationSeconds(s.ttl()),
	}); err != nil {
		slog.Warn("failed to store idempotency response", "plugin", plugin, "error", err)
	}
}

func (s StoreIdempotency) Abort(key string) {
	c, ok := s.client()
	if !ok {
		return
	}

	if err := c.Delete(idempotencyKeyPrefix + key); err != nil && !errors.Is(err, memcache.ErrCacheMiss) {
		slog.Warn("failed to abort idempotency entry", "plugin", plugin, "error", err)
	}
}

func (s StoreIdempotency) ttl() time.Duration {
	if s.TTL > 0 {
		return s.TTL
	}
	return defaultIdempotencyTTL
}

func (s StoreIdempotency) client() (*memcache.Client, bool) {
	name := s.Name
	if name == "" {
		name = "default"
	}
	c, ok := Client(name).Get()
	return c, ok && c != nil
}
