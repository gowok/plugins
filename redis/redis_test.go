package redis

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gowok/gowok"
	"github.com/ngamux/ngamux"
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

// withTestClients swaps the package connection registry for the duration of a
// test and restores it afterwards.
func withTestClients(t *testing.T, next map[string]*goredis.Client) {
	t.Helper()

	mutex.Lock()
	prev := clients
	clients = next
	mutex.Unlock()

	t.Cleanup(func() {
		mutex.Lock()
		clients = prev
		mutex.Unlock()
	})
}

// isolateClients runs a test with an empty connection registry and closes
// connections opened by the test before restoring the previous registry.
func isolateClients(t *testing.T) {
	t.Helper()

	mutex.Lock()
	prev := clients
	clients = make(map[string]*goredis.Client)
	mutex.Unlock()

	t.Cleanup(func() {
		Close()

		mutex.Lock()
		clients = prev
		mutex.Unlock()
	})
}

func withRedisConfig(t *testing.T, config map[string]any) {
	t.Helper()

	configMap := gowok.Config.Map()
	prev, had := configMap[plugin]
	if config == nil {
		delete(configMap, plugin)
	} else {
		configMap[plugin] = config
	}

	t.Cleanup(func() {
		if had {
			configMap[plugin] = prev
		} else {
			delete(configMap, plugin)
		}
	})
}

func redisDSN(mr *miniredis.Miniredis) string {
	return "redis://" + mr.Addr() + "/0"
}

func newTestRedisClient(t *testing.T, mr *miniredis.Miniredis) *goredis.Client {
	t.Helper()

	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	return client
}

// newIdempotencyTestStore starts an in-memory Redis server and points the
// default connection at it.
func newIdempotencyTestStore(t *testing.T) (*miniredis.Miniredis, StoreIdempotency) {
	t.Helper()

	mr := miniredis.RunT(t)
	withTestClients(t, map[string]*goredis.Client{"default": newTestRedisClient(t, mr)})

	return mr, StoreIdempotency{}
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

func TestConfigureOpensEnabledConnections(t *testing.T) {
	isolateClients(t)

	mr := miniredis.RunT(t)
	withRedisConfig(t, map[string]any{
		"default":   map[string]any{"enabled": true, "dsn": redisDSN(mr)},
		"secondary": map[string]any{"enabled": true, "dsn": redisDSN(mr)},
	})

	Configure(context.Background())

	want := []string{"default", "secondary"}
	if got := Names(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Names() = %v, want %v", got, want)
	}

	if err := Ping(); err != nil {
		t.Fatalf("Ping() = %v, want nil", err)
	}
	if err := Ping("secondary"); err != nil {
		t.Fatalf("Ping(secondary) = %v, want nil", err)
	}
}

func TestConfigureSkipsDisabledAndBrokenConnections(t *testing.T) {
	isolateClients(t)

	mr := miniredis.RunT(t)
	unreachable := miniredis.RunT(t)
	unreachableDSN := redisDSN(unreachable)
	unreachable.Close()

	withRedisConfig(t, map[string]any{
		"disabled":    map[string]any{"enabled": false, "dsn": redisDSN(mr)},
		"invalid":     map[string]any{"enabled": true, "dsn": "://not-a-valid-url"},
		"unreachable": map[string]any{"enabled": true, "dsn": unreachableDSN},
	})

	Configure(context.Background())

	if got := Names(); len(got) != 0 {
		t.Fatalf("Names() = %v, want no connections", got)
	}
}

func TestConfigureWithoutConfiguration(t *testing.T) {
	isolateClients(t)
	withRedisConfig(t, nil)

	Configure(context.Background())

	if got := Names(); len(got) != 0 {
		t.Fatalf("Names() = %v, want no connections", got)
	}
}

func TestConfigureClosesPreviousConnections(t *testing.T) {
	isolateClients(t)
	withRedisConfig(t, nil)

	stale := goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:1"})
	mutex.Lock()
	clients["stale"] = stale
	mutex.Unlock()

	Configure(context.Background())

	if ClientNoDefault("stale").IsPresent() {
		t.Fatal("connection from the previous configuration should be closed")
	}
}

func TestRegisterHealthReportsUpAndDown(t *testing.T) {
	isolateClients(t)

	mr := miniredis.RunT(t)
	withRedisConfig(t, map[string]any{
		"default": map[string]any{"enabled": true, "dsn": redisDSN(mr)},
	})

	Configure(context.Background())

	prevMux := gowok.Web.HttpServeMux
	gowok.Web.HttpServeMux = ngamux.NewHttpServeMux()
	t.Cleanup(func() { gowok.Web.HttpServeMux = prevMux })

	gowok.Health.Configure()

	check := func(want string) {
		t.Helper()

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/health/redis", nil)
		gowok.Web.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("health status = %d, want %d", w.Code, http.StatusOK)
		}
		if !strings.Contains(w.Body.String(), want) {
			t.Fatalf("health body = %q, want it to contain %q", w.Body.String(), want)
		}
	}

	check(`"UP"`)

	mr.SetError("redis is down")
	check(`"DOWN"`)
}

func TestCloseSkipsNilClient(t *testing.T) {
	withTestClients(t, map[string]*goredis.Client{"nil": nil})

	Close()

	if got := Names(); len(got) != 0 {
		t.Fatalf("Names() = %v, want no connections", got)
	}
}

func TestPingWithoutConnection(t *testing.T) {
	withTestClients(t, map[string]*goredis.Client{})

	if err := Ping(); err == nil {
		t.Fatal("Ping() without a connection should fail")
	}
	if err := Ping("missing"); err == nil {
		t.Fatal("Ping(missing) without a connection should fail")
	}
}

func TestPingReportsServerError(t *testing.T) {
	mr := miniredis.RunT(t)
	withTestClients(t, map[string]*goredis.Client{"default": newTestRedisClient(t, mr)})

	mr.SetError("redis is down")

	if err := Ping(); err == nil {
		t.Fatal("Ping() should fail when the server returns an error")
	}
}

func TestSetJSONAndGetJSONRoundTrip(t *testing.T) {
	mr := miniredis.RunT(t)
	withTestClients(t, map[string]*goredis.Client{"default": newTestRedisClient(t, mr)})

	in := map[string]string{"name": "gowok"}
	if err := SetJSON(context.Background(), "key", in, 0); err != nil {
		t.Fatalf("SetJSON() = %v, want nil", err)
	}

	var out map[string]string
	if err := GetJSON(context.Background(), "key", &out); err != nil {
		t.Fatalf("GetJSON() = %v, want nil", err)
	}
	if !reflect.DeepEqual(out, in) {
		t.Fatalf("GetJSON() = %v, want %v", out, in)
	}
}

func TestGetJSONReturnsMissingKeyError(t *testing.T) {
	mr := miniredis.RunT(t)
	withTestClients(t, map[string]*goredis.Client{"default": newTestRedisClient(t, mr)})

	var out map[string]string
	err := GetJSON(context.Background(), "missing", &out)
	if !errors.Is(err, goredis.Nil) {
		t.Fatalf("GetJSON() error = %v, want redis.Nil", err)
	}
}

func TestGetJSONReturnsUnmarshalError(t *testing.T) {
	mr := miniredis.RunT(t)
	withTestClients(t, map[string]*goredis.Client{"default": newTestRedisClient(t, mr)})

	if err := SetJSON(context.Background(), "key", "not-a-number", 0); err != nil {
		t.Fatalf("SetJSON() = %v, want nil", err)
	}

	var out int
	if err := GetJSON(context.Background(), "key", &out); err == nil {
		t.Fatal("GetJSON() should fail when the stored value cannot be decoded")
	}
}

func TestSetJSONReportsServerError(t *testing.T) {
	mr := miniredis.RunT(t)
	withTestClients(t, map[string]*goredis.Client{"default": newTestRedisClient(t, mr)})

	mr.SetError("redis is down")

	if err := SetJSON(context.Background(), "key", "value", 0); err == nil {
		t.Fatal("SetJSON() should fail when the server returns an error")
	}
}
