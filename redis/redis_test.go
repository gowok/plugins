package redis

import (
	"context"
	"errors"
	"testing"

	goredis "github.com/redis/go-redis/v9"
)

// injectClient registers an already-built client under name and removes it
// again when the test finishes. goredis clients are lazy, so no server is
// contacted unless a command is actually sent.
func injectClient(t *testing.T, name string, client *goredis.Client) {
	t.Helper()

	mutex.Lock()
	clients[name] = client
	mutex.Unlock()

	t.Cleanup(func() {
		mutex.Lock()
		delete(clients, name)
		mutex.Unlock()
		_ = client.Close()
	})
}

func TestNameOrDefault(t *testing.T) {
	if got := nameOrDefault(); got != "default" {
		t.Fatalf("nameOrDefault() = %q, want %q", got, "default")
	}

	if got := nameOrDefault("cache"); got != "cache" {
		t.Fatalf("nameOrDefault(\"cache\") = %q, want %q", got, "cache")
	}
}

func TestSetJSONWithoutConnection(t *testing.T) {
	err := SetJSON(context.Background(), "key", map[string]string{"a": "b"}, 0)
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("SetJSON error = %v, want ErrNotConfigured", err)
	}

	err = SetJSON(context.Background(), "key", "value", 0, "missing")
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("SetJSON error = %v, want ErrNotConfigured", err)
	}
}

func TestGetJSONWithoutConnection(t *testing.T) {
	var out map[string]string

	err := GetJSON(context.Background(), "key", &out)
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("GetJSON error = %v, want ErrNotConfigured", err)
	}
}

func TestSetJSONReportsMarshalError(t *testing.T) {
	injectClient(t, "default", goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:6379"}))

	err := SetJSON(context.Background(), "key", make(chan int), 0)
	if err == nil {
		t.Fatal("SetJSON accepted a non-marshalable value")
	}

	if errors.Is(err, ErrNotConfigured) {
		t.Fatalf("marshal error reported as ErrNotConfigured: %v", err)
	}
}

func TestNamesAndCloseOnEmptyState(t *testing.T) {
	if names := Names(); len(names) != 0 {
		t.Fatalf("Names() = %v, want empty", names)
	}

	Close()

	if names := Names(); len(names) != 0 {
		t.Fatalf("Names() after Close = %v, want empty", names)
	}
}

func TestClientNoDefaultEmpty(t *testing.T) {
	if _, ok := ClientNoDefault("missing").Get(); ok {
		t.Fatal("ClientNoDefault returned a connection for an unknown name")
	}
}

func TestConfigFromMap(t *testing.T) {
	configs := ConfigFromMap(map[string]any{
		"default": map[string]any{"enabled": true, "dsn": "redis://localhost:6379/0"},
		"cache":   map[string]any{"enabled": false, "dsn": "redis://localhost:6379/1"},
	})

	if len(configs) != 2 {
		t.Fatalf("ConfigFromMap returned %d configs, want 2", len(configs))
	}

	def, ok := configs["default"]
	if !ok {
		t.Fatal("ConfigFromMap dropped the default connection")
	}
	if !def.Enabled || def.DSN != "redis://localhost:6379/0" {
		t.Fatalf("ConfigFromMap(default) = %+v", def)
	}

	if cache := configs["cache"]; cache.Enabled || cache.DSN != "redis://localhost:6379/1" {
		t.Fatalf("ConfigFromMap(cache) = %+v", cache)
	}
}
