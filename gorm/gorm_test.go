package gorm

import (
	"testing"

	"github.com/gowok/gowok"
	"gorm.io/gorm"
)

func TestConfigureClearsConnectionsFromPreviousConfig(t *testing.T) {
	prevDBs := dbs
	dbs = dbMap{"stale": &gorm.DB{}}
	t.Cleanup(func() { dbs = prevDBs })

	config := gowok.Config.Map()
	prevConfig, hadConfig := config["gorm"]
	config["gorm"] = map[string]any{}
	t.Cleanup(func() {
		if hadConfig {
			config["gorm"] = prevConfig
		} else {
			delete(config, "gorm")
		}
	})

	Configure(nil)()

	if DBNoDefault("stale").IsPresent() {
		t.Fatal("connection from previous config should no longer be available")
	}
}

func TestDBNoDefaultWithoutName(t *testing.T) {
	prev := dbs
	dbs = dbMap{"default": &gorm.DB{}}
	t.Cleanup(func() { dbs = prev })

	if _, ok := DBNoDefault().Get(); !ok {
		t.Fatal("DBNoDefault() without name should return the default connection")
	}

	if _, ok := DBNoDefault("missing").Get(); ok {
		t.Fatal(`DBNoDefault("missing") should not fall back to the default connection`)
	}
}
