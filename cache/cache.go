package cache

import (
	"log/slog"

	"github.com/dgraph-io/ristretto"
	"github.com/eko/gocache/lib/v4/cache"
	"github.com/eko/gocache/lib/v4/store"
	store_redis "github.com/eko/gocache/store/redis/v4"
	store_memory "github.com/eko/gocache/store/ristretto/v4"
	"github.com/gowok/gowok"
	"github.com/gowok/gowok/must"
	"github.com/gowok/gowok/some"
	"github.com/redis/go-redis/v9"
)

var plugin = "cache"
var caches = map[string]store.StoreInterface{}

type C[T any] struct {
	*cache.Cache[T]
}

func Configure(project *gowok.Project) {
	config, err := ConfigFromProject(project)
	if err != nil {
		slog.Warn(err.Error(), "plugin", plugin)
		return
	}

	for name, dbC := range config {
		if !dbC.Enabled {
			continue
		}

		if dbC.Driver == "memory" {
			clientOpt := must.Must(ristretto.NewCache(&ristretto.Config{
				NumCounters: 1e7,
				MaxCost:     1 << 30,
				BufferItems: 64,
			}))
			caches[name] = store_memory.NewRistretto(clientOpt, store.WithSynchronousSet())
			return
		}

		if dbC.Driver == "redis" {
			clientOpt := must.Must(redis.ParseURL(dbC.DSN))
			caches[name] = store_redis.NewRedis(redis.NewClient(clientOpt))
			return
		}

		slog.Warn("unknown", "driver", dbC.Driver, "plugin", plugin)
	}
}

func Cache(name ...string) some.Some[*C[any]] {
	n := ""
	if len(name) > 0 {
		n = name[0]
		if db, ok := caches[n]; ok {
			c := cache.New[any](db)
			return some.Of(&C[any]{c})
		}
	}

	if n != "" {
		slog.Info("using default connection", "not_found", n)
	}

	if db, ok := caches["default"]; ok {
		c := cache.New[any](db)
		return some.Of(&C[any]{c})
	}

	return some.Empty[*C[any]]()
}
