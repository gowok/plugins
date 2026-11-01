package cache

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/eko/gocache/lib/v4/cache"
	"github.com/eko/gocache/lib/v4/store"
)

var plugin = "cache"

type C[T any] struct {
	*cache.Cache[T]
}

func (c *C[T]) GetSet(ctx context.Context, key string, loader func(context.Context) (T, error)) (T, error) {
	var zero T

	value, err := c.Get(ctx, key)
	if err == nil {
		return value, nil
	}
	if !errors.Is(err, &store.NotFound{}) {
		return zero, fmt.Errorf("cache get %q: %w", key, err)
	}

	if loader == nil {
		return zero, fmt.Errorf("cache loader for %q is nil", key)
	}

	value, err = loader(ctx)
	if err != nil {
		return zero, fmt.Errorf("cache loader %q: %w", key, err)
	}

	if err := c.Set(ctx, key, value); err != nil {
		slog.Warn("cache write-back failed", "key", key, "error", err, "plugin", plugin)
	}

	return value, nil
}
