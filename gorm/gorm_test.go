package gorm

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"slices"
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

func TestCloseDropsConnections(t *testing.T) {
	sqlDB := sql.OpenDB(stubConnector{})
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("failed to ping test database: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	prevDBs := dbs
	dbs = dbMap{"stale": {Config: &gorm.Config{ConnPool: sqlDB}}}
	t.Cleanup(func() { dbs = prevDBs })

	Close()

	if err := sqlDB.Ping(); err == nil {
		t.Fatal("Close should close the configured connection")
	}

	if DBNoDefault("stale").IsPresent() {
		t.Fatal("Close should forget the configured connection")
	}

	Close()
}

func TestNamesListsSortedConnections(t *testing.T) {
	prev := dbs
	dbs = dbMap{"replica": &gorm.DB{}, "default": &gorm.DB{}, "analytics": &gorm.DB{}}
	t.Cleanup(func() { dbs = prev })

	got := Names()
	want := []string{"analytics", "default", "replica"}
	if !slices.Equal(got, want) {
		t.Fatalf("Names() = %v, want %v", got, want)
	}
}

func TestNamesWithoutConnections(t *testing.T) {
	prev := dbs
	dbs = make(dbMap)
	t.Cleanup(func() { dbs = prev })

	if got := Names(); len(got) != 0 {
		t.Fatalf("Names() = %v, want no names", got)
	}
}

func TestPingConfiguredConnection(t *testing.T) {
	sqlDB := sql.OpenDB(stubConnector{})
	t.Cleanup(func() { _ = sqlDB.Close() })

	prev := dbs
	dbs = dbMap{"default": {Config: &gorm.Config{ConnPool: sqlDB}}}
	t.Cleanup(func() { dbs = prev })

	if err := Ping(); err != nil {
		t.Fatalf("Ping() returned an error: %v", err)
	}
}

func TestPingReturnsErrorWithoutConnection(t *testing.T) {
	prev := dbs
	dbs = make(dbMap)
	t.Cleanup(func() { dbs = prev })

	if err := Ping("missing"); err == nil {
		t.Fatal("Ping() should return an error when no connection is configured")
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

func TestRestoreClearsDeletedAt(t *testing.T) {
	db, err := gorm.Open(stubDialector{}, &gorm.Config{
		DryRun:                 true,
		DisableAutomaticPing:   true,
		SkipDefaultTransaction: true,
	})
	if err != nil {
		t.Fatalf("failed to open dry-run database: %v", err)
	}

	var built string
	var vars []any
	if err := db.Callback().Update().After("gorm:update").Register("test:capture_sql", func(tx *gorm.DB) {
		built = tx.Statement.SQL.String()
		vars = tx.Statement.Vars
	}); err != nil {
		t.Fatalf("failed to register callback: %v", err)
	}

	if err := Restore(db, &configuredKeyModel{}, "42"); err != nil {
		t.Fatalf("restore failed: %v", err)
	}

	if strings.Contains(built, "IS NULL") {
		t.Fatalf("restore must be unscoped so deleted rows are matched, got: %s", built)
	}

	if !strings.Contains(built, `"code" =`) {
		t.Fatalf("restore should filter by the model primary key, got: %s", built)
	}

	if len(vars) == 0 || vars[0] != nil {
		t.Fatalf("restore should null out deleted_at first, got vars: %v", vars)
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
