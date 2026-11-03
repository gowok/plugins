package gorm

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/gowok/fp/maps"
	"github.com/gowok/fp/some"
	"github.com/gowok/gowok"
	"github.com/gowok/gowok/web"
	"github.com/ngamux/ngamux"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

type Opener func(string) gorm.Dialector
type dbMap map[string]*gorm.DB

var plugin = "gorm"
var dbs = make(dbMap)

func Configure(drivers map[string]Opener, cfgs ...gorm.Option) func() {
	cfgs = append([]gorm.Option{&gorm.Config{Logger: logger.Default.LogMode(logger.Silent)}}, cfgs...)
	return func() {
		closeConnections()
		dbs = make(dbMap)

		var config Configs
		err := maps.ToStruct(maps.Get[map[string]any](gowok.Config.Map(), "gorm"), &config)
		if err != nil {
			slog.Warn("no configuration", "plugin", plugin, "error", err)
			return
		}

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
			gowok.Health.Add(healthName, func() any {
				var a int
				if err := db.Raw("select 1").Scan(&a).Error; err != nil {
					return ngamux.Map{"status": "DOWN"}
				}
				return ngamux.Map{"status": "UP"}
			})
		}
	}
}

func closeConnections() {
	for name, db := range dbs {
		if db == nil || db.Config == nil {
			continue
		}

		sqlDB, err := db.DB()
		if err != nil || sqlDB == nil {
			continue
		}

		if err := sqlDB.Close(); err != nil {
			slog.Warn("failed to close", "name", name, "plugin", plugin, "error", err)
		}
	}
}

func Close() {
	closeConnections()
	dbs = make(dbMap)
}

func Names() []string {
	names := make([]string, 0, len(dbs))
	for name := range dbs {
		names = append(names, name)
	}

	slices.Sort(names)
	return names
}

func Ping(name ...string) error {
	n := "default"
	if len(name) > 0 {
		n = name[0]
	}

	db, ok := DB(n).Get()
	if !ok || db == nil || db.Config == nil {
		return fmt.Errorf("gorm: no connection available for %q", n)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	return sqlDB.Ping()
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
	n := "default"
	if len(name) > 0 {
		n = name[0]
	}

	if db, ok := dbs[n]; ok {
		return some.Of(db)
	}

	return some.Empty[*gorm.DB]()
}

func FindByID[T any](db *gorm.DB, id any) (T, error) {
	var dest T
	err := db.Where(clause.Eq{Column: clause.PrimaryColumn, Value: id}).First(&dest).Error
	return dest, err
}

func Count[T any](db *gorm.DB, q ...func(*gorm.DB) *gorm.DB) (int64, error) {
	query := db.Model(new(T))
	for _, filter := range q {
		if filter != nil {
			query = filter(query)
		}
	}

	var count int64
	err := query.Count(&count).Error
	return count, err
}

func Exists[T any](db *gorm.DB, id any) (bool, error) {
	count, err := Count[T](db, func(d *gorm.DB) *gorm.DB {
		return d.Where(clause.Eq{Column: clause.PrimaryColumn, Value: id})
	})
	return count > 0, err
}

func Upsert[T any](db *gorm.DB, record *T) error {
	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(record); err != nil {
		return err
	}

	columns := make([]clause.Column, 0, len(stmt.Schema.PrimaryFields))
	for _, field := range stmt.Schema.PrimaryFields {
		columns = append(columns, clause.Column{Name: field.DBName})
	}
	if len(columns) == 0 {
		return fmt.Errorf("gorm: upsert requires a model with a primary key")
	}

	return db.Clauses(clause.OnConflict{Columns: columns, UpdateAll: true}).Create(record).Error
}

func UpdateByID[T any](db *gorm.DB, id any, values any) (T, error) {
	var dest T
	err := db.Model(dest).Where(clause.Eq{Column: clause.PrimaryColumn, Value: id}).Updates(values).Error
	if err != nil {
		return dest, err
	}

	return FindByID[T](db, id)
}

func DeleteByID[T any](db *gorm.DB, id any) error {
	var dest T
	return db.Unscoped().Where(clause.Eq{Column: clause.PrimaryColumn, Value: id}).Delete(&dest).Error
}

func SoftDelete[T any](db *gorm.DB, id string) error {
	var dest T
	return db.Model(dest).Where(clause.Eq{Column: clause.PrimaryColumn, Value: id}).Updates(struct {
		DeletedAt gorm.DeletedAt
	}{
		DeletedAt: gorm.DeletedAt{Time: time.Now(), Valid: true},
	}).Error
}

func Restore[T any](db *gorm.DB, id string) error {
	var dest T
	return db.Model(dest).Unscoped().Where(clause.Eq{Column: clause.PrimaryColumn, Value: id}).Updates(map[string]any{
		"deleted_at": nil,
	}).Error
}

func FindAllPaginated[T any](ctx context.Context, db *gorm.DB, pagination *web.Pagination[T], q ...func(*gorm.DB) *gorm.DB) (*web.Pagination[T], error) {
	if pagination == nil {
		pagination = &web.Pagination[T]{
			Page:    1,
			PerPage: 10,
			Filter:  map[string]any{},
			Sort:    map[string]string{},
			Data:    make([]T, 0),
		}
	}

	query := db.WithContext(ctx)

	for _, filter := range q {
		if filter != nil {
			query = filter(query)
		}
	}

	var count int64
	if err := query.Session(&gorm.Session{}).Model(new(T)).Count(&count).Error; err != nil {
		return nil, err
	}
	pagination.TotalRecord = int(count)

	if pagination.Page > 0 && pagination.PerPage > 0 {
		query = query.Offset(pagination.Skip()).Limit(pagination.PerPage)
	}

	sortColumns := make([]string, 0, len(pagination.Sort))
	for column := range pagination.Sort {
		sortColumns = append(sortColumns, column)
	}
	slices.Sort(sortColumns)
	for _, column := range sortColumns {
		direction := strings.ToLower(strings.TrimSpace(pagination.Sort[column]))
		query = query.Order(clause.OrderByColumn{
			Column: clause.Column{Name: column},
			Desc:   direction == "desc",
		})
	}

	if err := query.Find(&pagination.Data).Error; err != nil {
		return nil, err
	}

	return pagination, nil
}
