package memcached

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/gowok/fp/maps"
	"github.com/gowok/fp/some"
	"github.com/gowok/gowok"
	"github.com/gowok/gowok/cache"
	"github.com/ngamux/ngamux"
)

const healthCheckTimeout = 5 * time.Second

var plugin = "memcached"

var ErrNotConfigured = errors.New("memcached: no connection available")

var (
	mutex   sync.RWMutex
	clients = make(map[string]*memcache.Client)
)

func Configure(_ context.Context) {
	Close()

	configs := ConfigFromMap(maps.Get[map[string]any](gowok.Config.Map(), plugin))
	if len(configs) == 0 {
		slog.Warn("no configuration", "plugin", plugin)
		return
	}

	for name, dbC := range configs {
		if !dbC.Enabled {
			continue
		}

		c := memcache.New(dbC.Servers...)
		c.Timeout = healthCheckTimeout
		if err := c.Ping(); err != nil {
			_ = c.Close()
			slog.Warn("failed to connect", "name", name, "plugin", plugin, "error", err)
			continue
		}

		mutex.Lock()
		clients[name] = c
		mutex.Unlock()

		registerHealth(name, c)
		cache.Register(name, &client{c})
	}
}

func registerHealth(name string, c *memcache.Client) {
	healthName := plugin
	if name != "default" {
		healthName += "-" + name
	}

	gowok.Health.Add(healthName, func() any {
		if err := c.Ping(); err != nil {
			return ngamux.Map{"status": "DOWN"}
		}
		return ngamux.Map{"status": "UP"}
	})
}

func Close() {
	mutex.Lock()
	existing := clients
	clients = make(map[string]*memcache.Client)
	mutex.Unlock()

	for _, c := range existing {
		if c == nil {
			continue
		}
		if err := c.Close(); err != nil {
			slog.Warn("failed to close", "plugin", plugin, "error", err)
		}
	}
}

func Names() []string {
	mutex.RLock()
	names := make([]string, 0, len(clients))
	for name := range clients {
		names = append(names, name)
	}
	mutex.RUnlock()

	slices.Sort(names)
	return names
}

func Ping(name ...string) error {
	c, ok := Client(name...).Get()
	if !ok || c == nil {
		return fmt.Errorf("memcached: no connection available for %q", nameOrDefault(name...))
	}
	return c.Ping()
}

func SetJSON(ctx context.Context, key string, v any, expiration time.Duration, name ...string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c, ok := Client(name...).Get()
	if !ok || c == nil {
		return fmt.Errorf("%w for %q", ErrNotConfigured, nameOrDefault(name...))
	}

	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return c.Set(&memcache.Item{
		Key:        key,
		Value:      data,
		Expiration: expirationSeconds(expiration),
	})
}

func GetJSON(ctx context.Context, key string, v any, name ...string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c, ok := Client(name...).Get()
	if !ok || c == nil {
		return fmt.Errorf("%w for %q", ErrNotConfigured, nameOrDefault(name...))
	}

	item, err := c.Get(key)
	if err != nil {
		return err
	}
	return json.Unmarshal(item.Value, v)
}

func expirationSeconds(ttl time.Duration) int32 {
	if ttl <= 0 {
		return 0
	}

	const maxRelativeTTL = 30 * 24 * time.Hour
	if ttl > maxRelativeTTL {
		return int32(time.Now().Add(ttl).Unix())
	}

	seconds := int64((ttl + time.Second - 1) / time.Second)
	return int32(seconds)
}

func nameOrDefault(name ...string) string {
	if len(name) > 0 {
		return name[0]
	}
	return "default"
}

func Client(name ...string) some.Some[*memcache.Client] {
	n := nameOrDefault(name...)
	if c := ClientNoDefault(n); c.IsPresent() {
		return c
	}
	if n == "default" {
		return some.Empty[*memcache.Client]()
	}

	slog.Info("using default connection", "not_found", n, "plugin", plugin)
	if c := ClientNoDefault("default"); c.IsPresent() {
		return c
	}
	return some.Empty[*memcache.Client]()
}

func ClientNoDefault(name ...string) some.Some[*memcache.Client] {
	n := nameOrDefault(name...)

	mutex.RLock()
	c, ok := clients[n]
	mutex.RUnlock()
	if !ok || c == nil {
		return some.Empty[*memcache.Client]()
	}
	return some.Of(c)
}
