package redis

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/gowok/fp/maps"
	"github.com/gowok/fp/some"
	"github.com/gowok/gowok"
	"github.com/ngamux/ngamux"
	goredis "github.com/redis/go-redis/v9"
)

const healthCheckTimeout = 5 * time.Second

var plugin = "redis"

var (
	mutex   sync.RWMutex
	clients = make(map[string]*goredis.Client)
)

func Configure() {
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

		opts, err := goredis.ParseURL(dbC.DSN)
		if err != nil {
			slog.Warn("invalid DSN", "name", name, "plugin", plugin, "error", err)
			continue
		}

		client := goredis.NewClient(opts)

		ctx, cancel := context.WithTimeout(context.Background(), healthCheckTimeout)
		err = client.Ping(ctx).Err()
		cancel()
		if err != nil {
			_ = client.Close()
			slog.Warn("failed to connect", "name", name, "plugin", plugin, "error", err)
			continue
		}

		mutex.Lock()
		clients[name] = client
		mutex.Unlock()

		registerHealth(name, client)
	}
}

func registerHealth(name string, client *goredis.Client) {
	healthName := plugin
	if name != "default" {
		healthName += "-" + name
	}

	gowok.Health.Add(healthName, func() any {
		ctx, cancel := context.WithTimeout(context.Background(), healthCheckTimeout)
		defer cancel()

		if err := client.Ping(ctx).Err(); err != nil {
			return ngamux.Map{"status": "DOWN"}
		}
		return ngamux.Map{"status": "UP"}
	})
}

func Close() {
	mutex.Lock()
	existing := clients
	clients = make(map[string]*goredis.Client)
	mutex.Unlock()

	for name, client := range existing {
		if client == nil {
			continue
		}

		if err := client.Close(); err != nil {
			slog.Warn("failed to close", "name", name, "plugin", plugin, "error", err)
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
	n := "default"
	if len(name) > 0 {
		n = name[0]
	}

	client, ok := Client(n).Get()
	if !ok || client == nil {
		return fmt.Errorf("redis: no connection available for %q", n)
	}

	ctx, cancel := context.WithTimeout(context.Background(), healthCheckTimeout)
	defer cancel()

	return client.Ping(ctx).Err()
}

func Client(name ...string) some.Some[*goredis.Client] {
	n := "default"
	if len(name) > 0 {
		n = name[0]
	}

	if client := ClientNoDefault(n); client.IsPresent() {
		return client
	}

	if n == "default" {
		return some.Empty[*goredis.Client]()
	}

	slog.Info("using default connection", "not_found", n, "plugin", plugin)
	if client := ClientNoDefault("default"); client.IsPresent() {
		return client
	}

	return some.Empty[*goredis.Client]()
}

func ClientNoDefault(name ...string) some.Some[*goredis.Client] {
	n := "default"
	if len(name) > 0 {
		n = name[0]
	}

	mutex.RLock()
	client, ok := clients[n]
	mutex.RUnlock()

	if !ok || client == nil {
		return some.Empty[*goredis.Client]()
	}

	return some.Of(client)
}
