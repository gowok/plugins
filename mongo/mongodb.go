package mongo

import (
	"context"
	"log/slog"

	"github.com/gowok/gowok"
	"github.com/gowok/gowok/maps"
	"github.com/gowok/gowok/some"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var plugin = "mongo"

var mongos = make(map[string]*mongo.Client)

func Configure(project *gowok.Project) {
	var config Configs
	err := maps.ToStruct(maps.Get[map[string]any](project.ConfigMap, "mongo"), &config)
	if err != nil {
		slog.Warn("no configuration", "plugin", "mongo")
	}

	mongos = make(map[string]*mongo.Client)
	c := context.Background()

	for name, dbC := range config {
		if !dbC.Enabled {
			continue
		}

		opts := options.Client().ApplyURI(dbC.DSN)
		client, err := mongo.Connect(c, opts)
		if err != nil {
			slog.Warn("failed to open", "plugin", plugin, "name", name, "error", err)
			return
		}

		mongos[name] = client
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
