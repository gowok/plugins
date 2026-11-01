package gorm

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"testing"

	"github.com/gowok/gowok"
	"gorm.io/gorm"
)

type stubConnector struct{}

func (stubConnector) Connect(context.Context) (driver.Conn, error) { return stubConn{}, nil }
func (stubConnector) Driver() driver.Driver                        { return stubDriver{} }

type stubDriver struct{}

func (stubDriver) Open(string) (driver.Conn, error) { return stubConn{}, nil }

type stubConn struct{}

func (stubConn) Prepare(string) (driver.Stmt, error) { return nil, driver.ErrSkip }
func (stubConn) Close() error                        { return nil }
func (stubConn) Begin() (driver.Tx, error)           { return nil, driver.ErrSkip }

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

func TestConfigureClosesConnectionsFromPreviousConfig(t *testing.T) {
	sqlDB := sql.OpenDB(stubConnector{})
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("failed to ping test database: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	prevDBs := dbs
	dbs = dbMap{"stale": {Config: &gorm.Config{ConnPool: sqlDB}}}
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

	if err := sqlDB.Ping(); err == nil {
		t.Fatal("connection from previous configuration should be closed")
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
