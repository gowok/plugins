package gorm

import (
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/ngamux/middleware/idempotency"
	"gorm.io/driver/sqlite"
	gormlib "gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newIdempotencyGormDB(t *testing.T) *gormlib.DB {
	t.Helper()

	db, err := gormlib.Open(sqlite.Open(":memory:"), &gormlib.Config{
		Logger:               logger.Default.LogMode(logger.Silent),
		DisableAutomaticPing: true,
	})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get test database handle: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)

	prev := dbs
	dbs = dbMap{"default": db}
	t.Cleanup(func() {
		dbs = prev
		_ = sqlDB.Close()
	})

	return db
}

func TestGormStoreIdempotencyBeginStartsRequest(t *testing.T) {
	db := newIdempotencyGormDB(t)
	store := StoreIdempotency{}

	response, state := store.Begin("k")
	if state != idempotency.StateMissing {
		t.Fatalf("Begin() state = %v, want StateMissing", state)
	}
	if response != nil {
		t.Fatalf("Begin() response = %#v, want nil", response)
	}

	var record idempotencyRecord
	if err := db.Table(store.tableName()).Where("idempotency_key = ?", "k").Take(&record).Error; err != nil {
		t.Fatalf("in-progress entry was not stored: %v", err)
	}
	if !record.InProgress {
		t.Fatal("stored entry should be marked as in progress")
	}
}

func TestGormStoreIdempotencyBeginWhileInProgress(t *testing.T) {
	newIdempotencyGormDB(t)
	store := StoreIdempotency{}

	if _, state := store.Begin("k"); state != idempotency.StateMissing {
		t.Fatalf("first Begin() state = %v, want StateMissing", state)
	}

	response, state := store.Begin("k")
	if state != idempotency.StateInProgress {
		t.Fatalf("second Begin() state = %v, want StateInProgress", state)
	}
	if response != nil {
		t.Fatalf("second Begin() response = %#v, want nil", response)
	}
}

func TestGormStoreIdempotencyCompleteThenFound(t *testing.T) {
	newIdempotencyGormDB(t)
	store := StoreIdempotency{}

	if _, state := store.Begin("k"); state != idempotency.StateMissing {
		t.Fatalf("first Begin() state = %v, want StateMissing", state)
	}

	want := &idempotency.StoredResponse{
		Status: http.StatusCreated,
		Header: http.Header{
			"Content-Type": []string{"text/plain"},
			"X-Test":       []string{"yes"},
		},
		Body: []byte("hello"),
	}
	store.Complete("k", want)

	got, state := store.Begin("k")
	if state != idempotency.StateFound {
		t.Fatalf("Begin() after Complete() state = %v, want StateFound", state)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Begin() after Complete() = %#v, want %#v", got, want)
	}
}

func TestGormStoreIdempotencyAbortRemovesEntry(t *testing.T) {
	newIdempotencyGormDB(t)
	store := StoreIdempotency{}

	if _, state := store.Begin("k"); state != idempotency.StateMissing {
		t.Fatalf("first Begin() state = %v, want StateMissing", state)
	}

	store.Abort("k")

	if _, state := store.Begin("k"); state != idempotency.StateMissing {
		t.Fatalf("Begin() after Abort() state = %v, want StateMissing", state)
	}
}

func TestGormStoreIdempotencyCompleteNilAborts(t *testing.T) {
	newIdempotencyGormDB(t)
	store := StoreIdempotency{}

	if _, state := store.Begin("k"); state != idempotency.StateMissing {
		t.Fatalf("first Begin() state = %v, want StateMissing", state)
	}

	store.Complete("k", nil)

	if _, state := store.Begin("k"); state != idempotency.StateMissing {
		t.Fatalf("Begin() after Complete(nil) state = %v, want StateMissing", state)
	}
}

func TestGormStoreIdempotencyExpiredResponseCanBeClaimed(t *testing.T) {
	db := newIdempotencyGormDB(t)
	store := StoreIdempotency{TTL: time.Second}

	if _, state := store.Begin("k"); state != idempotency.StateMissing {
		t.Fatalf("first Begin() state = %v, want StateMissing", state)
	}
	store.Complete("k", &idempotency.StoredResponse{Status: http.StatusOK, Body: []byte("ok")})

	if _, state := store.Begin("k"); state != idempotency.StateFound {
		t.Fatalf("Begin() before expiry state = %v, want StateFound", state)
	}

	expired := time.Now().Add(-time.Second).Unix()
	if err := db.Table(store.tableName()).
		Where("idempotency_key = ?", "k").
		Update("expires_at", expired).Error; err != nil {
		t.Fatalf("failed to expire entry: %v", err)
	}

	if _, state := store.Begin("k"); state != idempotency.StateMissing {
		t.Fatalf("Begin() after expiry state = %v, want StateMissing", state)
	}
}

func TestGormStoreIdempotencyCustomTable(t *testing.T) {
	db := newIdempotencyGormDB(t)
	store := StoreIdempotency{Table: "custom_idempotency"}

	if _, state := store.Begin("k"); state != idempotency.StateMissing {
		t.Fatalf("Begin() state = %v, want StateMissing", state)
	}

	if !db.Migrator().HasTable("custom_idempotency") {
		t.Fatal("custom idempotency table was not created")
	}
}

func TestGormStoreIdempotencyWithoutConnection(t *testing.T) {
	prev := dbs
	dbs = dbMap{}
	t.Cleanup(func() { dbs = prev })

	store := StoreIdempotency{}

	response, state := store.Begin("k")
	if state != idempotency.StateMissing {
		t.Fatalf("Begin() without a connection state = %v, want StateMissing", state)
	}
	if response != nil {
		t.Fatalf("Begin() without a connection response = %#v, want nil", response)
	}

	store.Complete("k", &idempotency.StoredResponse{Status: http.StatusOK})
	store.Abort("k")
}

func TestGormStoreIdempotencyUsesNamedConnection(t *testing.T) {
	db := newIdempotencyGormDB(t)
	prev := dbs
	dbs = dbMap{"cache": db}
	t.Cleanup(func() { dbs = prev })

	store := StoreIdempotency{Name: "cache"}
	if _, state := store.Begin("k"); state != idempotency.StateMissing {
		t.Fatalf("Begin() state = %v, want StateMissing", state)
	}

	var record idempotencyRecord
	if err := db.Table(store.tableName()).Where("idempotency_key = ?", "k").Take(&record).Error; err != nil {
		t.Fatalf("in-progress entry was not stored in the named connection: %v", err)
	}
}
