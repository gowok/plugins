package memcached

import (
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/ngamux/middleware/idempotency"
)

func newIdempotencyTestStore(t *testing.T) (*fakeMemcached, StoreIdempotency) {
	t.Helper()
	server := startFakeMemcached(t)
	withTestClients(t, map[string]*memcache.Client{"default": newTestMemcacheClient(t, server)})
	return server, StoreIdempotency{}
}

func TestStoreIdempotencyBeginStartsRequest(t *testing.T) {
	server, store := newIdempotencyTestStore(t)
	response, state := store.Begin("key")
	if state != idempotency.StateMissing || response != nil {
		t.Fatalf("Begin() = %#v, %v; want nil, StateMissing", response, state)
	}
	item, ok := server.item(idempotencyKeyPrefix + "key")
	if !ok || string(item.value) != idempotencyInProgress {
		t.Fatalf("stored item = %#v, %v; want in-progress entry", item, ok)
	}
	if !item.expiration.IsZero() {
		t.Fatalf("in-progress expiration = %v, want none", item.expiration)
	}
}

func TestStoreIdempotencyBeginWhileInProgress(t *testing.T) {
	_, store := newIdempotencyTestStore(t)
	if _, state := store.Begin("key"); state != idempotency.StateMissing {
		t.Fatalf("first Begin() state = %v, want StateMissing", state)
	}
	response, state := store.Begin("key")
	if state != idempotency.StateInProgress || response != nil {
		t.Fatalf("second Begin() = %#v, %v; want nil, StateInProgress", response, state)
	}
}

func TestStoreIdempotencyCompleteThenFound(t *testing.T) {
	_, store := newIdempotencyTestStore(t)
	if _, state := store.Begin("key"); state != idempotency.StateMissing {
		t.Fatalf("Begin() state = %v, want StateMissing", state)
	}
	want := &idempotency.StoredResponse{
		Status: http.StatusCreated,
		Header: http.Header{"Content-Type": []string{"text/plain"}},
		Body:   []byte("hello"),
	}
	store.Complete("key", want)

	got, state := store.Begin("key")
	if state != idempotency.StateFound {
		t.Fatalf("Begin() after Complete() state = %v, want StateFound", state)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Begin() after Complete() = %#v, want %#v", got, want)
	}
}

func TestStoreIdempotencyAbortRemovesEntry(t *testing.T) {
	_, store := newIdempotencyTestStore(t)
	if _, state := store.Begin("key"); state != idempotency.StateMissing {
		t.Fatalf("Begin() state = %v, want StateMissing", state)
	}
	store.Abort("key")
	if _, state := store.Begin("key"); state != idempotency.StateMissing {
		t.Fatalf("Begin() after Abort() state = %v, want StateMissing", state)
	}
}

func TestStoreIdempotencyCompleteNilAborts(t *testing.T) {
	_, store := newIdempotencyTestStore(t)
	if _, state := store.Begin("key"); state != idempotency.StateMissing {
		t.Fatalf("Begin() state = %v, want StateMissing", state)
	}
	store.Complete("key", nil)
	if _, state := store.Begin("key"); state != idempotency.StateMissing {
		t.Fatalf("Begin() after Complete(nil) state = %v, want StateMissing", state)
	}
}

func TestStoreIdempotencyCompletedResponseExpires(t *testing.T) {
	server, store := newIdempotencyTestStore(t)
	store.TTL = time.Minute
	if _, state := store.Begin("key"); state != idempotency.StateMissing {
		t.Fatalf("Begin() state = %v, want StateMissing", state)
	}
	store.Complete("key", &idempotency.StoredResponse{Status: http.StatusOK, Body: []byte("ok")})
	item, ok := server.item(idempotencyKeyPrefix + "key")
	if !ok || item.expiration.IsZero() {
		t.Fatalf("completed item = %#v, %v; want expiring entry", item, ok)
	}

	server.expire(idempotencyKeyPrefix + "key")
	if _, state := store.Begin("key"); state != idempotency.StateMissing {
		t.Fatalf("Begin() after expiry state = %v, want StateMissing", state)
	}
}

func TestStoreIdempotencyWithoutConnection(t *testing.T) {
	withTestClients(t, map[string]*memcache.Client{})
	store := StoreIdempotency{}
	response, state := store.Begin("key")
	if state != idempotency.StateMissing || response != nil {
		t.Fatalf("Begin() = %#v, %v; want nil, StateMissing", response, state)
	}
	store.Complete("key", &idempotency.StoredResponse{Status: http.StatusOK})
	store.Abort("key")
}

func TestStoreIdempotencyBeginReportsServerError(t *testing.T) {
	server, store := newIdempotencyTestStore(t)
	server.setError("memcached is down")
	response, state := store.Begin("key")
	if state != idempotency.StateMissing || response != nil {
		t.Fatalf("Begin() = %#v, %v; want nil, StateMissing", response, state)
	}
}

func TestStoreIdempotencyBeginRejectsCorruptEntry(t *testing.T) {
	server, store := newIdempotencyTestStore(t)
	server.setRaw(idempotencyKeyPrefix+"key", []byte("not-json"))
	response, state := store.Begin("key")
	if state != idempotency.StateMissing || response != nil {
		t.Fatalf("Begin() = %#v, %v; want nil, StateMissing", response, state)
	}
}

func TestStoreIdempotencyCompleteReportsServerError(t *testing.T) {
	server, store := newIdempotencyTestStore(t)
	if _, state := store.Begin("key"); state != idempotency.StateMissing {
		t.Fatalf("Begin() state = %v, want StateMissing", state)
	}
	server.setError("memcached is down")
	store.Complete("key", &idempotency.StoredResponse{Status: http.StatusOK})
}

func TestStoreIdempotencyAbortReportsServerError(t *testing.T) {
	server, store := newIdempotencyTestStore(t)
	if _, state := store.Begin("key"); state != idempotency.StateMissing {
		t.Fatalf("Begin() state = %v, want StateMissing", state)
	}
	server.setError("memcached is down")
	store.Abort("key")
}

func TestStoreIdempotencyAbortMissingKey(t *testing.T) {
	_, store := newIdempotencyTestStore(t)
	store.Abort("missing")
}

func TestStoreIdempotencyUsesNamedConnection(t *testing.T) {
	server := startFakeMemcached(t)
	withTestClients(t, map[string]*memcache.Client{"cache": newTestMemcacheClient(t, server)})
	store := StoreIdempotency{Name: "cache"}
	if _, state := store.Begin("key"); state != idempotency.StateMissing {
		t.Fatalf("Begin() state = %v, want StateMissing", state)
	}
	if _, ok := server.item(idempotencyKeyPrefix + "key"); !ok {
		t.Fatal("named connection did not store the idempotency entry")
	}
}

func TestStoreIdempotencyTTL(t *testing.T) {
	store := StoreIdempotency{}
	if got := store.ttl(); got != defaultIdempotencyTTL {
		t.Fatalf("ttl() = %v, want %v", got, defaultIdempotencyTTL)
	}
	store.TTL = time.Minute
	if got := store.ttl(); got != time.Minute {
		t.Fatalf("ttl() = %v, want %v", got, time.Minute)
	}
}

func TestStoreIdempotencyAddConflictButMissingEntry(t *testing.T) {
	server, store := newIdempotencyTestStore(t)
	server.mu.Lock()
	server.items[idempotencyKeyPrefix+"key"] = fakeMemcachedItem{
		value:      []byte(idempotencyInProgress),
		expiration: time.Now().Add(-time.Second),
	}
	server.mu.Unlock()

	response, state := store.Begin("key")
	if state != idempotency.StateMissing || response != nil {
		t.Fatalf("Begin() = %#v, %v; want nil, StateMissing", response, state)
	}
}

func TestStoreIdempotencyGetServerErrorAfterAddConflict(t *testing.T) {
	server, store := newIdempotencyTestStore(t)
	if _, state := store.Begin("key"); state != idempotency.StateMissing {
		t.Fatalf("Begin() state = %v, want StateMissing", state)
	}
	server.setError("memcached is down")
	response, state := store.Begin("key")
	if state != idempotency.StateMissing || response != nil {
		t.Fatalf("Begin() = %#v, %v; want nil, StateMissing", response, state)
	}
}
