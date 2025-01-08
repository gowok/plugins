package mongo

import (
	"context"
	"log/slog"

	"github.com/gowok/gowok"
	"github.com/gowok/gowok/some"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mongoDBMap map[string]*mongo.Client

var mongos = make(mongoDBMap)

func Configure(project *gowok.Project) {
	configAny, ok := project.ConfigMap["mongo"]
	if !ok {
		slog.Warn("no configuration", "plugin", "mongo")
		return
	}
	configMap, ok := configAny.(map[string]any)
	if !ok {
		slog.Warn("no configuration", "plugin", "mongo")
		return
	}
	config := ConfigFromMap(configMap)

	mongos = make(mongoDBMap)
	c := context.Background()

	for name, dbC := range config {
		if !dbC.Enabled {
			continue
		}

		opts := options.Client().ApplyURI(dbC.DSN)
		client, err := mongo.Connect(c, opts)
		if err != nil {
			slog.Warn("failed to open mongo", "name", name, "error", err)
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

	db := ClientNoDefault(n)
	if db.IsPresent() {
		return db
	}

	slog.Info("using default connection", "not_found", name)
	db = ClientNoDefault("default")
	if db.IsPresent() {
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
