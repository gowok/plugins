package cache

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/eko/gocache/lib/v4/cache"
	"github.com/eko/gocache/lib/v4/store"
	store_redis "github.com/eko/gocache/store/redis/v4"
	store_memory "github.com/eko/gocache/store/ristretto/v4"
	"github.com/gowok/fp/some"
	"github.com/gowok/gowok"
	"github.com/gowok/gowok/async"
	"github.com/ngamux/ngamux"
	"github.com/redis/go-redis/v9"
)

var plugin = "cache"
var caches = sync.Map{}
var redisClients = sync.Map{}

// healthCheckTimeout bounds the Redis probe so an unhealthy server is reported
// as DOWN instead of blocking the health endpoint indefinitely.
const healthCheckTimeout = 5 * time.Second

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

				redisClient := redis.NewClient(clientOpt)
				if pingErr := redisClient.Ping(context.Background()).Err(); pingErr != nil {
					_ = redisClient.Close()
					return nil, fmt.Errorf("redis unreachable for %q: %w", name, pingErr)
				}

				caches.Store(name, store_redis.NewRedis(redisClient))
				redisClients.Store(name, redisClient)
				return
			}

			return nil, fmt.Errorf("unknown driver %q for %q", dbC.Driver, name)
		})
	}

	_, err = async.All(tasks...)
	if err != nil {
		slog.Error(err.Error(), "plugin", plugin)
	}

	registerHealth(config)
}

func registerHealth(config Configs) {
	for name, dbC := range config {
		if !dbC.Enabled {
			continue
		}

		if _, ok := caches.Load(name); !ok {
			continue
		}

		healthName := "cache"
		if name != "default" {
			healthName += "-" + name
		}

		if dbC.Driver != "redis" {
			gowok.Health.Add(healthName, func() any {
				return ngamux.Map{"status": "UP"}
			})
			continue
		}

		clientAny, ok := redisClients.Load(name)
		if !ok {
			continue
		}

		redisClient, ok := clientAny.(*redis.Client)
		if !ok {
			continue
		}

		gowok.Health.Add(healthName, func() any {
			ctx, cancel := context.WithTimeout(context.Background(), healthCheckTimeout)
			defer cancel()

			if err := redisClient.Ping(ctx).Err(); err != nil {
				return ngamux.Map{"status": "DOWN"}
			}
			return ngamux.Map{"status": "UP"}
		})
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
			slog.Warn("cache backend has unexpected type", "name", n, "type", fmt.Sprintf("%T", dbAny), "plugin", plugin)
		}
	}

	if n != "" {
		slog.Warn("cache not found, falling back to default", "not_found", n, "plugin", plugin)
	}

	if dbAny, ok := caches.Load("default"); ok {
		if db, ok := dbAny.(store.StoreInterface); ok {
			c := cache.New[T](db)
			return some.Of(&C[T]{c})
		}
		slog.Warn("cache backend has unexpected type", "name", "default", "type", fmt.Sprintf("%T", dbAny), "plugin", plugin)
	}

	slog.Warn("no cache available", "name", n, "plugin", plugin)
	return some.Empty[*C[T]]()
}
