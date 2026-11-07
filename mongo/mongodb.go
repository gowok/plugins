package mongo

import (
	"context"
	"log/slog"
	"time"

	"github.com/gowok/fp/maps"
	"github.com/gowok/fp/some"
	"github.com/gowok/gowok"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var plugin = "mongo"

const closeTimeout = 5 * time.Second

var mongos = make(map[string]*mongo.Client)

func Configure() {
	var config Configs
	err := maps.ToStruct(maps.Get[map[string]any](gowok.Config.Map(), "mongo"), &config)
	if err != nil {
		slog.Warn("no configuration", "plugin", "mongo")
	}

	Close()
	c := context.Background()

	for name, dbC := range config {
		if !dbC.Enabled {
			continue
		}

		opts := options.Client().ApplyURI(dbC.DSN)
		client, err := mongo.Connect(c, opts)
		if err != nil {
			slog.Warn("failed to open", "plugin", plugin, "name", name, "error", err)
			continue
		}

		mongos[name] = client
	}
}

func Close() {
	clients := mongos
	mongos = make(map[string]*mongo.Client)

	ctx, cancel := context.WithTimeout(context.Background(), closeTimeout)
	defer cancel()

	for name, client := range clients {
		if client == nil {
			continue
		}

		if err := client.Disconnect(ctx); err != nil {
			slog.Warn("failed to close", "name", name, "plugin", plugin, "error", err)
		}
	}
}

func Client(name ...string) some.Some[*mongo.Client] {
	n := "default"
	if len(name) > 0 {
		n = name[0]
	}

	if db := ClientNoDefault(n); db.IsPresent() {
		return db
	}

	if n == "default" {
		return some.Empty[*mongo.Client]()
	}

	slog.Info("using default connection", "not_found", n)
	if db := ClientNoDefault("default"); db.IsPresent() {
		return db
	}

	return some.Empty[*mongo.Client]()
}

func ClientNoDefault(name ...string) some.Some[*mongo.Client] {
	n := ""
	if len(name) > 0 {
		n = name[0]
		if db, ok := mongos[n]; ok {
			return some.Of(db)
		}
	}

	return some.Empty[*mongo.Client]()
}
