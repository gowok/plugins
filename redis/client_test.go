package redis

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

func newClientForTest(t *testing.T) (*client, *miniredis.Miniredis) {
	t.Helper()

	mr := miniredis.RunT(t)
	return &client{redisClient: newTestRedisClient(t, mr)}, mr
}

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
			c, _ := newClientForTest(t)
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
	c, _ := newClientForTest(t)

	if err := c.Set(context.Background(), "key", make(chan int)); err == nil {
		t.Fatal("Set() accepted a value that cannot be encoded")
	}
}

func TestClientSetReportsRedisError(t *testing.T) {
	c, mr := newClientForTest(t)
	mr.SetError("redis is down")

	if err := c.Set(context.Background(), "key", 123); err == nil {
		t.Fatal("Set() did not report the Redis error")
	}
}

func TestClientGetMissingKey(t *testing.T) {
	c, _ := newClientForTest(t)

	got, err := c.Get(context.Background(), "missing")
	if !errors.Is(err, goredis.Nil) {
		t.Fatalf("Get() error = %v, want redis.Nil", err)
	}
	if got != nil {
		t.Fatalf("Get() = %#v, want nil", got)
	}
}

func TestClientGetReportsRedisError(t *testing.T) {
	c, mr := newClientForTest(t)
	mr.SetError("redis is down")

	got, err := c.Get(context.Background(), "key")
	if err == nil {
		t.Fatal("Get() did not report the Redis error")
	}
	if got != nil {
		t.Fatalf("Get() = %#v, want nil", got)
	}
}

func TestClientGetReportsDecodeError(t *testing.T) {
	c, mr := newClientForTest(t)
	if err := mr.Set("key", "not-gob-data"); err != nil {
		t.Fatal(err)
	}

	got, err := c.Get(context.Background(), "key")
	if err == nil {
		t.Fatal("Get() did not report invalid gob data")
	}
	if got != nil {
		t.Fatalf("Get() = %#v, want nil", got)
	}
}

func TestClientDelete(t *testing.T) {
	c, _ := newClientForTest(t)
	ctx := context.Background()

	if err := c.Set(ctx, "key", 123); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if err := c.Delete(ctx, "key"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := c.Get(ctx, "key"); !errors.Is(err, goredis.Nil) {
		t.Fatalf("Get() after Delete error = %v, want redis.Nil", err)
	}
}

func TestClientDeleteMissingKey(t *testing.T) {
	c, _ := newClientForTest(t)

	if err := c.Delete(context.Background(), "missing"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
}

func TestClientDeleteReportsRedisError(t *testing.T) {
	c, mr := newClientForTest(t)
	mr.SetError("redis is down")

	if err := c.Delete(context.Background(), "key"); err == nil {
		t.Fatal("Delete() did not report the Redis error")
	}
}
