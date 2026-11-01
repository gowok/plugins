package gorm

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"slices"
	"strings"
	"testing"

	"github.com/gowok/gowok"
	"github.com/gowok/gowok/web"
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

	if err := SoftDelete[configuredKeyModel](db, "42"); err != nil {
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

	if err := Restore[configuredKeyModel](db, "42"); err != nil {
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

func TestFindByIDUsesModelPrimaryKey(t *testing.T) {
	db, err := gorm.Open(stubDialector{}, &gorm.Config{
		DryRun:                 true,
		DisableAutomaticPing:   true,
		SkipDefaultTransaction: true,
	})
	if err != nil {
		t.Fatalf("failed to open dry-run database: %v", err)
	}

	var built string
	if err := db.Callback().Query().After("gorm:query").Register("test:capture_sql", func(tx *gorm.DB) {
		built = tx.Statement.SQL.String()
	}); err != nil {
		t.Fatalf("failed to register callback: %v", err)
	}

	if _, err := FindByID[configuredKeyModel](db, "42"); err != nil {
		t.Fatalf("find by id failed: %v", err)
	}

	if !strings.Contains(built, `"code" =`) {
		t.Fatalf("find by id should filter by the model primary key, got: %s", built)
	}
}

func TestExistsUsesModelPrimaryKey(t *testing.T) {
	db, err := gorm.Open(stubDialector{}, &gorm.Config{
		DryRun:                 true,
		DisableAutomaticPing:   true,
		SkipDefaultTransaction: true,
	})
	if err != nil {
		t.Fatalf("failed to open dry-run database: %v", err)
	}

	var built string
	if err := db.Callback().Query().After("gorm:query").Register("test:capture_sql", func(tx *gorm.DB) {
		built = tx.Statement.SQL.String()
	}); err != nil {
		t.Fatalf("failed to register callback: %v", err)
	}

	if _, err := Exists[configuredKeyModel](db, "42"); err != nil {
		t.Fatalf("exists failed: %v", err)
	}

	if !strings.Contains(built, "count(*)") {
		t.Fatalf("exists should count matching rows, got: %s", built)
	}

	if !strings.Contains(built, `"code" =`) {
		t.Fatalf("exists should filter by the model primary key, got: %s", built)
	}
}

func TestUpdateByIDUsesModelPrimaryKey(t *testing.T) {
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

	if _, err := UpdateByID[configuredKeyModel](db, "42", map[string]any{"name": "renamed"}); err != nil {
		t.Fatalf("update by id failed: %v", err)
	}

	if !strings.Contains(built, `"name"=`) && !strings.Contains(built, `"name" =`) {
		t.Fatalf("update by id should set the requested column, got: %s", built)
	}

	if !strings.Contains(built, `"code" =`) {
		t.Fatalf("update by id should filter by the model primary key, got: %s", built)
	}
}

func TestUpdateByIDAcceptsStruct(t *testing.T) {
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

	if _, err := UpdateByID[configuredKeyModel](db, "42", configuredKeyModel{Name: "renamed"}); err != nil {
		t.Fatalf("update by id failed: %v", err)
	}

	if !strings.Contains(built, `"name"=`) && !strings.Contains(built, `"name" =`) {
		t.Fatalf("update by id should set the requested column, got: %s", built)
	}

	if !strings.Contains(built, `"code" =`) {
		t.Fatalf("update by id should filter by the model primary key, got: %s", built)
	}
}

func TestDeleteByIDIsUnscoped(t *testing.T) {
	db, err := gorm.Open(stubDialector{}, &gorm.Config{
		DryRun:                 true,
		DisableAutomaticPing:   true,
		SkipDefaultTransaction: true,
	})
	if err != nil {
		t.Fatalf("failed to open dry-run database: %v", err)
	}

	var built string
	if err := db.Callback().Delete().After("gorm:delete").Register("test:capture_sql", func(tx *gorm.DB) {
		built = tx.Statement.SQL.String()
	}); err != nil {
		t.Fatalf("failed to register callback: %v", err)
	}

	if err := DeleteByID[configuredKeyModel](db, "42"); err != nil {
		t.Fatalf("delete by id failed: %v", err)
	}

	if !strings.Contains(built, `"code" =`) {
		t.Fatalf("delete by id should filter by the model primary key, got: %s", built)
	}

	if strings.Contains(built, "IS NULL") {
		t.Fatalf("delete by id must be unscoped to remove the row, got: %s", built)
	}
}

func TestFindAllPaginatedCountsBeforePaging(t *testing.T) {
	db, err := gorm.Open(stubDialector{}, &gorm.Config{
		DryRun:                 true,
		DisableAutomaticPing:   true,
		SkipDefaultTransaction: true,
	})
	if err != nil {
		t.Fatalf("failed to open dry-run database: %v", err)
	}

	type executedQuery struct {
		sql  string
		vars []any
	}
	var queries []executedQuery
	if err := db.Callback().Query().After("gorm:query").Register("test:capture_sql", func(tx *gorm.DB) {
		queries = append(queries, executedQuery{tx.Statement.SQL.String(), append([]any(nil), tx.Statement.Vars...)})
	}); err != nil {
		t.Fatalf("failed to register callback: %v", err)
	}

	pagination := &web.Pagination[configuredKeyModel]{
		Page:    3,
		PerPage: 5,
		Filter:  map[string]any{"name": "renamed"},
	}
	got, err := FindAllPaginated(context.Background(), db, pagination, func(q *gorm.DB) *gorm.DB {
		return q.Where(clause.Eq{Column: "code", Value: "42"})
	})
	if err != nil {
		t.Fatalf("find all paginated failed: %v", err)
	}

	if got != pagination {
		t.Fatal("find all paginated should report through the given pagination")
	}

	if len(queries) != 2 {
		t.Fatalf("expected a count query and a page query, got %d: %+v", len(queries), queries)
	}

	countQuery, pageQuery := queries[0], queries[1]
	if !strings.Contains(countQuery.sql, "count(*)") {
		t.Fatalf("first query should count matching rows, got: %s", countQuery.sql)
	}

	if strings.Contains(countQuery.sql, "LIMIT") {
		t.Fatalf("count query must run before paging, got: %s", countQuery.sql)
	}

	for _, condition := range []string{`"name" =`, `"code" =`} {
		if !strings.Contains(countQuery.sql, condition) {
			t.Fatalf("count query should apply the %s condition, got: %s", condition, countQuery.sql)
		}
	}

	if !strings.Contains(pageQuery.sql, "LIMIT") || !strings.Contains(pageQuery.sql, "OFFSET") {
		t.Fatalf("page query should be limited and offset, got: %s", pageQuery.sql)
	}

	if len(pageQuery.vars) < 2 {
		t.Fatalf("page query should bind the offset and limit, got vars: %v", pageQuery.vars)
	}

	limit, offset := pageQuery.vars[len(pageQuery.vars)-2], pageQuery.vars[len(pageQuery.vars)-1]
	if limit != 5 || offset != 10 {
		t.Fatalf("page query should limit to 5 and offset by (3-1)*5=10, got limit %v and offset %v", limit, offset)
	}

	queries = nil
	if _, err := FindAllPaginated(context.Background(), db, &web.Pagination[configuredKeyModel]{}); err != nil {
		t.Fatalf("find all paginated without paging failed: %v", err)
	}

	if len(queries) != 2 {
		t.Fatalf("expected a count query and a page query, got %d: %+v", len(queries), queries)
	}

	if strings.Contains(queries[1].sql, "LIMIT") || strings.Contains(queries[1].sql, "OFFSET") {
		t.Fatalf("page 0 should not be limited nor offset, got: %s", queries[1].sql)
	}
}

func TestFindAllPaginatedAppliesSort(t *testing.T) {
	db, err := gorm.Open(stubDialector{}, &gorm.Config{
		DryRun:                 true,
		DisableAutomaticPing:   true,
		SkipDefaultTransaction: true,
	})
	if err != nil {
		t.Fatalf("failed to open dry-run database: %v", err)
	}

	var pageSQL string
	if err := db.Callback().Query().After("gorm:query").Register("test:capture_sql", func(tx *gorm.DB) {
		if strings.Contains(tx.Statement.SQL.String(), "count(*)") {
			return
		}
		pageSQL = tx.Statement.SQL.String()
	}); err != nil {
		t.Fatalf("failed to register callback: %v", err)
	}

	cases := []struct {
		name string
		sort map[string]string
		want string
		desc bool
	}{
		{"descending", map[string]string{"name": "desc"}, `ORDER BY "name"`, true},
		{"ascending", map[string]string{"name": "asc"}, `ORDER BY "name"`, false},
		{"unknown direction falls back to ascending", map[string]string{"name": "sideways"}, `ORDER BY "name"`, false},
		{"multiple columns sort deterministically", map[string]string{"name": "desc", "code": "asc"}, `ORDER BY "code","name"`, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pageSQL = ""
			pagination := &web.Pagination[configuredKeyModel]{
				Page:    1,
				PerPage: 10,
				Filter:  map[string]any{},
				Sort:    tc.sort,
			}
			if _, err := FindAllPaginated(context.Background(), db, pagination); err != nil {
				t.Fatalf("find all paginated failed: %v", err)
			}

			if !strings.Contains(pageSQL, tc.want) {
				t.Fatalf("page query should be ordered by %q, got: %s", tc.want, pageSQL)
			}

			if desc := strings.Contains(pageSQL, "DESC"); desc != tc.desc {
				t.Fatalf("page query descending state should be %v, got: %s", tc.desc, pageSQL)
			}
		})
	}
}

func TestFindAllPaginatedDefaultsNilPagination(t *testing.T) {
	db, err := gorm.Open(stubDialector{}, &gorm.Config{
		DryRun:                 true,
		DisableAutomaticPing:   true,
		SkipDefaultTransaction: true,
	})
	if err != nil {
		t.Fatalf("failed to open dry-run database: %v", err)
	}

	type executedQuery struct {
		sql  string
		vars []any
	}
	var queries []executedQuery
	if err := db.Callback().Query().After("gorm:query").Register("test:capture_sql", func(tx *gorm.DB) {
		queries = append(queries, executedQuery{tx.Statement.SQL.String(), append([]any(nil), tx.Statement.Vars...)})
	}); err != nil {
		t.Fatalf("failed to register callback: %v", err)
	}

	got, err := FindAllPaginated[configuredKeyModel](context.Background(), db, nil)
	if err != nil {
		t.Fatalf("find all paginated with nil pagination failed: %v", err)
	}

	if got == nil {
		t.Fatal("find all paginated should return an initialized pagination")
	}

	if got.Page != 1 || got.PerPage != 10 {
		t.Fatalf("nil pagination should default to page 1 and 10 per page, got page %d and per page %d", got.Page, got.PerPage)
	}

	if got.Filter == nil || got.Sort == nil {
		t.Fatalf("nil pagination should get empty filter and sort maps, got filter %v and sort %v", got.Filter, got.Sort)
	}

	if len(got.Filter) != 0 || len(got.Sort) != 0 {
		t.Fatalf("defaulted filter and sort should be empty, got filter %v and sort %v", got.Filter, got.Sort)
	}

	if got.Data == nil {
		t.Fatal("nil pagination should get an initialized data slice")
	}

	if len(queries) != 2 {
		t.Fatalf("expected a count query and a page query, got %d: %+v", len(queries), queries)
	}

	pageQuery := queries[1]
	if !strings.Contains(pageQuery.sql, "LIMIT") {
		t.Fatalf("defaulted pagination should still page, got: %s", pageQuery.sql)
	}

	if strings.Contains(pageQuery.sql, "OFFSET") {
		t.Fatalf("offset 0 should not be rendered, got: %s", pageQuery.sql)
	}

	if len(pageQuery.vars) == 0 {
		t.Fatalf("page query should bind the limit, got vars: %v", pageQuery.vars)
	}

	if limit := pageQuery.vars[len(pageQuery.vars)-1]; limit != 10 {
		t.Fatalf("defaulted pagination should limit to 10, got limit %v", limit)
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
