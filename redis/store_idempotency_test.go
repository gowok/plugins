package redis

import (
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/ngamux/middleware/idempotency"
	goredis "github.com/redis/go-redis/v9"
)

func TestStoreIdempotencyBeginStartsRequest(t *testing.T) {
	mr, store := newIdempotencyTestStore(t)

	response, state := store.Begin("k")
	if state != idempotency.StateMissing {
		t.Fatalf("Begin() state = %v, want StateMissing", state)
	}
	if response != nil {
		t.Fatalf("Begin() response = %#v, want nil", response)
	}

	raw, err := mr.Get(idempotencyKeyPrefix + "k")
	if err != nil {
		t.Fatalf("in-progress key was not stored: %v", err)
	}
	if raw != idempotencyInProgress {
		t.Fatalf("in-progress key = %q, want %q", raw, idempotencyInProgress)
	}
}

func TestStoreIdempotencyBeginWhileInProgress(t *testing.T) {
	_, store := newIdempotencyTestStore(t)

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

func TestStoreIdempotencyCompleteThenFound(t *testing.T) {
	_, store := newIdempotencyTestStore(t)

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

func TestStoreIdempotencyAbortRemovesEntry(t *testing.T) {
	_, store := newIdempotencyTestStore(t)

	if _, state := store.Begin("k"); state != idempotency.StateMissing {
		t.Fatalf("first Begin() state = %v, want StateMissing", state)
	}

	store.Abort("k")

	if _, state := store.Begin("k"); state != idempotency.StateMissing {
		t.Fatalf("Begin() after Abort() state = %v, want StateMissing", state)
	}
}

func TestStoreIdempotencyCompleteNilAborts(t *testing.T) {
	_, store := newIdempotencyTestStore(t)

	if _, state := store.Begin("k"); state != idempotency.StateMissing {
		t.Fatalf("first Begin() state = %v, want StateMissing", state)
	}

	store.Complete("k", nil)

	if _, state := store.Begin("k"); state != idempotency.StateMissing {
		t.Fatalf("Begin() after Complete(nil) state = %v, want StateMissing", state)
	}
}

func TestStoreIdempotencyInProgressDoesNotExpire(t *testing.T) {
	mr, store := newIdempotencyTestStore(t)
	store.TTL = time.Minute

	if _, state := store.Begin("k"); state != idempotency.StateMissing {
		t.Fatalf("first Begin() state = %v, want StateMissing", state)
	}

	mr.FastForward(2 * time.Minute)

	if _, state := store.Begin("k"); state != idempotency.StateInProgress {
		t.Fatalf("Begin() after TTL elapsed state = %v, want StateInProgress", state)
	}
}

func TestStoreIdempotencyCompletedResponseExpires(t *testing.T) {
	mr, store := newIdempotencyTestStore(t)
	store.TTL = time.Minute

	if _, state := store.Begin("k"); state != idempotency.StateMissing {
		t.Fatalf("first Begin() state = %v, want StateMissing", state)
	}
	store.Complete("k", &idempotency.StoredResponse{Status: http.StatusOK, Body: []byte("ok")})

	if _, state := store.Begin("k"); state != idempotency.StateFound {
		t.Fatalf("Begin() before expiry state = %v, want StateFound", state)
	}

	mr.FastForward(2 * time.Minute)

	if _, state := store.Begin("k"); state != idempotency.StateMissing {
		t.Fatalf("Begin() after expiry state = %v, want StateMissing", state)
	}
}

func TestStoreIdempotencyWithoutConnection(t *testing.T) {
	withTestClients(t, map[string]*goredis.Client{})

	store := StoreIdempotency{}

	response, state := store.Begin("k")
	if state != idempotency.StateMissing {
		t.Fatalf("Begin() without a connection state = %v, want StateMissing", state)
	}
	if response != nil {
		t.Fatalf("Begin() without a connection response = %#v, want nil", response)
	}

	// Must not panic when no connection is configured.
	store.Complete("k", &idempotency.StoredResponse{Status: http.StatusOK})
	store.Abort("k")
}

func TestStoreIdempotencyBeginReportsServerError(t *testing.T) {
	mr, store := newIdempotencyTestStore(t)
	mr.SetError("redis is down")

	response, state := store.Begin("k")
	if state != idempotency.StateMissing {
		t.Fatalf("Begin() state = %v, want StateMissing", state)
	}
	if response != nil {
		t.Fatalf("Begin() response = %#v, want nil", response)
	}
}

func TestStoreIdempotencyBeginRejectsCorruptEntry(t *testing.T) {
	mr, store := newIdempotencyTestStore(t)

	if err := mr.Set(idempotencyKeyPrefix+"k", "not-json"); err != nil {
		t.Fatalf("failed to seed entry: %v", err)
	}

	response, state := store.Begin("k")
	if state != idempotency.StateMissing {
		t.Fatalf("Begin() state = %v, want StateMissing", state)
	}
	if response != nil {
		t.Fatalf("Begin() response = %#v, want nil", response)
	}
}

func TestStoreIdempotencyCompleteReportsServerError(t *testing.T) {
	mr, store := newIdempotencyTestStore(t)

	if _, state := store.Begin("k"); state != idempotency.StateMissing {
		t.Fatalf("Begin() state = %v, want StateMissing", state)
	}

	mr.SetError("redis is down")

	// Must not panic when the server rejects the write.
	store.Complete("k", &idempotency.StoredResponse{Status: http.StatusOK, Body: []byte("ok")})
}

func TestStoreIdempotencyAbortReportsServerError(t *testing.T) {
	mr, store := newIdempotencyTestStore(t)

	if _, state := store.Begin("k"); state != idempotency.StateMissing {
		t.Fatalf("Begin() state = %v, want StateMissing", state)
	}

	mr.SetError("redis is down")

	// Must not panic when the server rejects the delete.
	store.Abort("k")
}

func TestStoreIdempotencyUsesNamedConnection(t *testing.T) {
	mr := miniredis.RunT(t)
	withTestClients(t, map[string]*goredis.Client{"cache": newTestRedisClient(t, mr)})

	store := StoreIdempotency{Name: "cache"}
	if _, state := store.Begin("k"); state != idempotency.StateMissing {
		t.Fatalf("Begin() state = %v, want StateMissing", state)
	}

	if _, err := mr.Get(idempotencyKeyPrefix + "k"); err != nil {
		t.Fatalf("in-progress key was not stored in the named connection: %v", err)
	}
}
