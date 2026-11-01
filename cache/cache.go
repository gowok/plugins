package cache

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/dgraph-io/ristretto"
	"github.com/eko/gocache/lib/v4/cache"
	"github.com/eko/gocache/lib/v4/store"
	store_redis "github.com/eko/gocache/store/redis/v4"
	store_memory "github.com/eko/gocache/store/ristretto/v4"
	"github.com/gowok/fp/some"
	"github.com/gowok/gowok"
	"github.com/gowok/gowok/async"
	"github.com/redis/go-redis/v9"
)

var plugin = "cache"
var caches = sync.Map{}

// map[string]store.StoreInterface{}

type C[T any] struct {
	*cache.Cache[T]
}

func estimateCost(value any) int64 {
	switch v := value.(type) {
	case string:
		if len(v) == 0 {
			return 1
		}
		return int64(len(v))
	case []byte:
		if len(v) == 0 {
			return 1
		}
		return int64(len(v))
	}
	return 1
}

func Configure() {
	config, err := ConfigFromProject()
	if err != nil {
		slog.Warn(err.Error(), "plugin", plugin)
		return
	}

	tasks := make([]func() (any, error), 0)
	for name, dbC := range config {
		if !dbC.Enabled {
			continue
		}

		tasks = append(tasks, func() (res any, err error) {
			if dbC.Driver == "memory" {
				clientOpt := gowok.Must(ristretto.NewCache(&ristretto.Config{NumCounters: 1e7,
					MaxCost:     1 << 30,
					BufferItems: 64,
					Cost:        estimateCost,
				}))
				caches.Store(name, store_memory.NewRistretto(clientOpt, store.WithSynchronousSet()))
				return
			}

			if dbC.Driver == "redis" {
				clientOpt, parseErr := redis.ParseURL(dbC.DSN)
				if parseErr != nil {
					return nil, fmt.Errorf("invalid redis dsn for %q: %w", name, parseErr)
				}

				caches.Store(name, store_redis.NewRedis(redis.NewClient(clientOpt)))
				return
			}

			slog.Warn("unknown", "driver", dbC.Driver, "plugin", plugin)
			return
		})
	}

	_, err = async.All(tasks...)
	if err != nil {
		slog.Error(err.Error(), "plugin", plugin)
	}
}

func Cache[T any](name ...string) some.Some[*C[T]] {
	n := ""
	if len(name) > 0 {
		n = name[0]
		if dbAny, ok := caches.Load(n); ok {
			if db, ok := dbAny.(store.StoreInterface); ok {
				c := cache.New[T](db)
				return some.Of(&C[T]{c})
			}
		}
	}

	if n != "" {
		slog.Info("using default connection", "not_found", n)
	}

	if dbAny, ok := caches.Load("default"); ok {
		if db, ok := dbAny.(store.StoreInterface); ok {
			c := cache.New[T](db)
			return some.Of(&C[T]{c})
		}
	}

	return some.Empty[*C[T]]()
}
