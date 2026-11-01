package gorm

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"strings"
	"testing"

	"github.com/gowok/gowok"
	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
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

type stubDialector struct{}

func (stubDialector) Name() string { return "stub" }
func (stubDialector) Initialize(db *gorm.DB) error {
	callbacks.RegisterDefaultCallbacks(db, &callbacks.Config{})
	return nil
}
func (stubDialector) Migrator(*gorm.DB) gorm.Migrator                { return nil }
func (stubDialector) DataTypeOf(*schema.Field) string                { return "" }
func (stubDialector) DefaultValueOf(*schema.Field) clause.Expression { return nil }
func (stubDialector) BindVarTo(clause.Writer, *gorm.Statement, any)  {}
func (stubDialector) QuoteTo(writer clause.Writer, name string) {
	_, _ = writer.WriteString(`"` + name + `"`)
}
func (stubDialector) Explain(sql string, vars ...any) string { return sql }

type configuredKeyModel struct {
	Code      string `gorm:"primaryKey"`
	Name      string
	DeletedAt gorm.DeletedAt
}

func TestSoftDeleteUsesModelPrimaryKey(t *testing.T) {
	db, err := gorm.Open(stubDialector{}, &gorm.Config{
		DryRun:                 true,
		DisableAutomaticPing:   true,
		SkipDefaultTransaction: true,
	})
	if err != nil {
		t.Fatalf("failed to open dry-run database: %v", err)
	}

	var built string
	if err := db.Callback().Update().After("gorm:update").Register("test:capture_sql", func(tx *gorm.DB) {
		built = tx.Statement.SQL.String()
	}); err != nil {
		t.Fatalf("failed to register callback: %v", err)
	}

	if err := SoftDelete(db, &configuredKeyModel{}, "42"); err != nil {
		t.Fatalf("soft delete failed: %v", err)
	}

	if !strings.Contains(built, `"code" =`) {
		t.Fatalf("soft delete should filter by the model primary key, got: %s", built)
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
