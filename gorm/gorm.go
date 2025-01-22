package gorm

import (
	"log/slog"

	"github.com/gowok/gowok"
	"github.com/gowok/gowok/health"
	"github.com/gowok/gowok/some"
	"github.com/ngamux/ngamux"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Opener func(string) gorm.Dialector
type dbMap map[string]*gorm.DB

var plugin = "gorm"
var dbs = make(dbMap)

func Configure(drivers map[string]Opener, cfgs ...gorm.Option) func(*gowok.Project) {
	cfgs = append([]gorm.Option{&gorm.Config{Logger: logger.Default.LogMode(logger.Silent)}}, cfgs...)
	return func(project *gowok.Project) {
		configAny, ok := project.ConfigMap["gorm"]
		if !ok {
			slog.Warn("no configuration", "plugin", plugin)
			return
		}
		configMap, ok := configAny.(map[string]any)
		if !ok {
			slog.Warn("no configuration", "plugin", plugin)
			return
		}
		config := ConfigFromMap(configMap)

		for name, dbC := range config {
			if !dbC.Enabled {
				continue
			}

			opener, ok := drivers[dbC.Driver]
			if !ok {
				slog.Warn("unknown GORM", "driver", dbC.Driver, "name", name, "plugin", plugin)
				continue
			}

			db, err := gorm.Open(opener(dbC.DSN), cfgs...)
			if err != nil {
				slog.Warn("failed to open", "driver", dbC.Driver, "name", name, "plugin", plugin, "error", err)
				continue
			}

			dbs[name] = db

			healthName := "gorm"
			if name != "default" {
				healthName += "-" + name
			}
			health.Add(healthName, func() any {
				var a int
				if err := db.Raw("select 1").Scan(&a).Error; err != nil {
					return ngamux.Map{"status": "DOWN"}
				}
				return ngamux.Map{"status": "UP"}
			})
		}
	}
}

func DB(name ...string) some.Some[*gorm.DB] {
	n := "default"
	if len(name) > 0 {
		n = name[0]
	}

	if db := DBNoDefault(n); db.IsPresent() {
		return db
	}

	if n == "default" {
		return some.Empty[*gorm.DB]()
	}

	slog.Info("using default connection", "not_found", n)
	if db := DBNoDefault("default"); db.IsPresent() {
		return db
	}
	return some.Empty[*gorm.DB]()
}

func DBNoDefault(name ...string) some.Some[*gorm.DB] {
	n := ""
	if len(name) > 0 {
		n = name[0]
		if db, ok := dbs[n]; ok {
			return some.Of(db)
		}
	}

	return some.Empty[*gorm.DB]()
}
