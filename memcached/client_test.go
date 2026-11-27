package memcached

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/gowok/gowok/cache"
)

var _ cache.Cache = (*client)(nil)

func TestClientSetGetRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		value any
	}{
		{name: "int", value: 123},
		{name: "string", value: "value"},
		{name: "bool", value: true},
		{name: "float", value: 1.5},
		{name: "bytes", value: []byte{0x00, 0x01, 0xfe, 0xff}},
		{name: "slice", value: []int{1, 2, 3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := startFakeMemcached(t)
			c := &client{memcacheClient: newTestMemcacheClient(t, server)}
			ctx := context.Background()

			if err := c.Set(ctx, "key", tt.value); err != nil {
				t.Fatalf("Set() error = %v", err)
			}
			got, err := c.Get(ctx, "key")
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.value) {
				t.Fatalf("Get() = %#v (%T), want %#v (%T)", got, got, tt.value, tt.value)
			}
		})
	}
}

func TestClientSetReportsEncodingError(t *testing.T) {
	server := startFakeMemcached(t)
	c := &client{memcacheClient: newTestMemcacheClient(t, server)}
	if err := c.Set(context.Background(), "key", make(chan int)); err == nil {
		t.Fatal("Set() accepted a value that cannot be encoded")
	}
}

func TestClientSetReportsMemcachedError(t *testing.T) {
	server := startFakeMemcached(t)
	c := &client{memcacheClient: newTestMemcacheClient(t, server)}
	server.setError("memcached is down")
	if err := c.Set(context.Background(), "key", 123); err == nil {
		t.Fatal("Set() did not report the memcached error")
	}
}

func TestClientHonorsCanceledContext(t *testing.T) {
	server := startFakeMemcached(t)
	c := &client{memcacheClient: newTestMemcacheClient(t, server)}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := c.Set(ctx, "key", 123); !errors.Is(err, context.Canceled) {
		t.Fatalf("Set() error = %v, want context.Canceled", err)
	}
	if got, err := c.Get(ctx, "key"); !errors.Is(err, context.Canceled) || got != nil {
		t.Fatalf("Get() = %#v, %v; want nil, context.Canceled", got, err)
	}
	if err := c.Delete(ctx, "key"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Delete() error = %v, want context.Canceled", err)
	}
}

func TestClientGetMissingKey(t *testing.T) {
	server := startFakeMemcached(t)
	c := &client{memcacheClient: newTestMemcacheClient(t, server)}
	got, err := c.Get(context.Background(), "missing")
	if !errors.Is(err, memcache.ErrCacheMiss) {
		t.Fatalf("Get() error = %v, want memcache.ErrCacheMiss", err)
	}
	if got != nil {
		t.Fatalf("Get() = %#v, want nil", got)
	}
}

func TestClientGetReportsMemcachedError(t *testing.T) {
	server := startFakeMemcached(t)
	c := &client{memcacheClient: newTestMemcacheClient(t, server)}
	server.setError("memcached is down")
	got, err := c.Get(context.Background(), "key")
	if err == nil {
		t.Fatal("Get() did not report the memcached error")
	}
	if got != nil {
		t.Fatalf("Get() = %#v, want nil", got)
	}
}

func TestClientGetReportsDecodeError(t *testing.T) {
	server := startFakeMemcached(t)
	c := &client{memcacheClient: newTestMemcacheClient(t, server)}
	server.setRaw("key", []byte("not-gob-data"))

	got, err := c.Get(context.Background(), "key")
	if err == nil {
		t.Fatal("Get() did not report invalid gob data")
	}
	if got != nil {
		t.Fatalf("Get() = %#v, want nil", got)
	}
}

func TestClientDelete(t *testing.T) {
	server := startFakeMemcached(t)
	c := &client{memcacheClient: newTestMemcacheClient(t, server)}
	ctx := context.Background()

	if err := c.Set(ctx, "key", 123); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if err := c.Delete(ctx, "key"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := c.Get(ctx, "key"); !errors.Is(err, memcache.ErrCacheMiss) {
		t.Fatalf("Get() after Delete error = %v, want memcache.ErrCacheMiss", err)
	}
}

func TestClientDeleteMissingKey(t *testing.T) {
	server := startFakeMemcached(t)
	c := &client{memcacheClient: newTestMemcacheClient(t, server)}
	if err := c.Delete(context.Background(), "missing"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
}

func TestClientDeleteReportsMemcachedError(t *testing.T) {
	server := startFakeMemcached(t)
	c := &client{memcacheClient: newTestMemcacheClient(t, server)}
	server.setError("memcached is down")
	if err := c.Delete(context.Background(), "key"); err == nil {
		t.Fatal("Delete() did not report the memcached error")
	}
}
