package openapi

import (
	"database/sql"
	"reflect"
	"time"

	"github.com/go-openapi/spec"
	gowok_sql "github.com/gowok/gowok/sql"
)

var sqlNull = map[reflect.Type]*spec.Schema{
	reflect.TypeOf(time.Time{}):                 spec.StringProperty(),
	reflect.TypeOf(sql.NullBool{}):              spec.BoolProperty(),
	reflect.TypeOf(sql.NullInt16{}):             spec.Int16Property(),
	reflect.TypeOf(sql.NullInt32{}):             spec.Int32Property(),
	reflect.TypeOf(sql.NullInt64{}):             spec.Int64Property(),
	reflect.TypeOf(sql.NullFloat64{}):           spec.Float64Property(),
	reflect.TypeOf(sql.NullString{}):            spec.StringProperty(),
	reflect.TypeOf(sql.NullByte{}):              spec.StringProperty(),
	reflect.TypeOf(sql.NullTime{}):              spec.StringProperty(),
	reflect.TypeOf(gowok_sql.Null[any]{}):       spec.StringProperty(),
	reflect.TypeOf(gowok_sql.Null[string]{}):    spec.StringProperty(),
	reflect.TypeOf(gowok_sql.Null[time.Time]{}): spec.StringProperty(),
	reflect.TypeOf(gowok_sql.Null[bool]{}):      spec.BoolProperty(),
	reflect.TypeOf(gowok_sql.Null[int16]{}):     spec.Int16Property(),
	reflect.TypeOf(gowok_sql.Null[int32]{}):     spec.Int32Property(),
	reflect.TypeOf(gowok_sql.Null[int64]{}):     spec.Int64Property(),
	reflect.TypeOf(gowok_sql.Null[float64]{}):   spec.Float64Property(),
}

var S = sqlNull
