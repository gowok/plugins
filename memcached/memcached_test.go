package memcached

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/gowok/gowok"
	gowokcache "github.com/gowok/gowok/cache"
	"github.com/ngamux/ngamux"
)

type fakeMemcachedItem struct {
	value      []byte
	flags      uint32
	expiration time.Time
}

type fakeMemcached struct {
	listener  net.Listener
	mu        sync.Mutex
	items     map[string]fakeMemcachedItem
	conns     map[net.Conn]struct{}
	serverErr error
	wg        sync.WaitGroup
	closeOnce sync.Once
}

func startFakeMemcached(t *testing.T) *fakeMemcached {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := &fakeMemcached{
		listener: listener,
		items:    make(map[string]fakeMemcachedItem),
		conns:    make(map[net.Conn]struct{}),
	}
	s.wg.Add(1)
	go s.accept()

	t.Cleanup(func() {
		s.closeOnce.Do(func() {
			_ = listener.Close()
			s.mu.Lock()
			for conn := range s.conns {
				_ = conn.Close()
			}
			s.mu.Unlock()
			s.wg.Wait()
		})
	})
	return s
}

func (s *fakeMemcached) accept() {
	defer s.wg.Done()
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		s.mu.Lock()
		s.conns[conn] = struct{}{}
		s.mu.Unlock()
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			defer func() {
				_ = conn.Close()
				s.mu.Lock()
				delete(s.conns, conn)
				s.mu.Unlock()
			}()
			s.serve(conn)
		}()
	}
}

func (s *fakeMemcached) serve(conn net.Conn) {
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		switch fields[0] {
		case "quit":
			return
		case "version":
			s.reply(writer)
		case "get", "gets":
			s.handleGet(writer, fields)
		case "set", "add":
			if !s.handleWrite(reader, writer, fields) {
				return
			}
		case "delete":
			s.handleDelete(writer, fields)
		default:
			_, _ = writer.WriteString("ERROR\r\n")
			_ = writer.Flush()
		}
	}
}

func (s *fakeMemcached) reply(writer *bufio.Writer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.serverErr != nil {
		_, _ = writer.WriteString("SERVER_ERROR injected\r\n")
	} else {
		_, _ = writer.WriteString("VERSION fake-memcached\r\n")
	}
	_ = writer.Flush()
}

func (s *fakeMemcached) handleGet(writer *bufio.Writer, fields []string) {
	if len(fields) != 2 {
		_, _ = writer.WriteString("CLIENT_ERROR bad get\r\n")
		_ = writer.Flush()
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.serverErr != nil {
		_, _ = writer.WriteString("SERVER_ERROR injected\r\n")
		_ = writer.Flush()
		return
	}

	key := fields[1]
	item, ok := s.items[key]
	if ok && !item.expiration.IsZero() && !item.expiration.After(time.Now()) {
		delete(s.items, key)
		ok = false
	}
	if !ok {
		_, _ = writer.WriteString("END\r\n")
		_ = writer.Flush()
		return
	}

	_, _ = fmt.Fprintf(writer, "VALUE %s %d %d\r\n", key, item.flags, len(item.value))
	_, _ = writer.Write(item.value)
	_, _ = writer.WriteString("\r\nEND\r\n")
	_ = writer.Flush()
}

func (s *fakeMemcached) handleWrite(reader *bufio.Reader, writer *bufio.Writer, fields []string) bool {
	if len(fields) < 5 {
		_, _ = writer.WriteString("CLIENT_ERROR bad write\r\n")
		_ = writer.Flush()
		return true
	}

	flags, _ := strconv.ParseUint(fields[2], 10, 32)
	expiration, _ := strconv.ParseInt(fields[3], 10, 32)
	size, _ := strconv.Atoi(fields[4])
	body := make([]byte, size+2)
	if _, err := io.ReadFull(reader, body); err != nil {
		return false
	}
	if !bytes.HasSuffix(body, []byte("\r\n")) {
		return false
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.serverErr != nil {
		_, _ = writer.WriteString("SERVER_ERROR injected\r\n")
		_ = writer.Flush()
		return true
	}

	key := fields[1]
	existing, exists := s.items[key]
	if exists && !existing.expiration.IsZero() && !existing.expiration.After(time.Now()) {
		delete(s.items, key)
		exists = false
	}
	if fields[0] == "add" && exists {
		_, _ = writer.WriteString("NOT_STORED\r\n")
		_ = writer.Flush()
		return true
	}

	s.items[key] = fakeMemcachedItem{
		value:      append([]byte(nil), body[:size]...),
		flags:      uint32(flags),
		expiration: fakeExpiration(int32(expiration)),
	}
	_, _ = writer.WriteString("STORED\r\n")
	_ = writer.Flush()
	return true
}

func (s *fakeMemcached) handleDelete(writer *bufio.Writer, fields []string) {
	if len(fields) != 2 {
		_, _ = writer.WriteString("CLIENT_ERROR bad delete\r\n")
		_ = writer.Flush()
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.serverErr != nil {
		_, _ = writer.WriteString("SERVER_ERROR injected\r\n")
		_ = writer.Flush()
		return
	}

	key := fields[1]
	item, ok := s.items[key]
	if ok && !item.expiration.IsZero() && !item.expiration.After(time.Now()) {
		delete(s.items, key)
		ok = false
	}
	if !ok {
		_, _ = writer.WriteString("NOT_FOUND\r\n")
	} else {
		delete(s.items, key)
		_, _ = writer.WriteString("DELETED\r\n")
	}
	_ = writer.Flush()
}

func fakeExpiration(expiration int32) time.Time {
	if expiration == 0 {
		return time.Time{}
	}
	if expiration < 0 {
		return time.Now().Add(-time.Second)
	}
	if int64(expiration) > 30*24*60*60 {
		return time.Unix(int64(expiration), 0)
	}
	return time.Now().Add(time.Duration(expiration) * time.Second)
}

func (s *fakeMemcached) addr() string {
	return s.listener.Addr().String()
}

func (s *fakeMemcached) setError(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if message == "" {
		s.serverErr = nil
		return
	}
	s.serverErr = errors.New(message)
}

func (s *fakeMemcached) setRaw(key string, value []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[key] = fakeMemcachedItem{value: append([]byte(nil), value...)}
}

func (s *fakeMemcached) expire(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[key]
	if !ok {
		return
	}
	item.expiration = time.Now().Add(-time.Second)
	s.items[key] = item
}

func (s *fakeMemcached) item(key string) (fakeMemcachedItem, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[key]
	return item, ok
}

func newTestMemcacheClient(t *testing.T, server *fakeMemcached) *memcache.Client {
	t.Helper()
	c := memcache.New(server.addr())
	c.Timeout = time.Second
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func withTestClients(t *testing.T, next map[string]*memcache.Client) {
	t.Helper()
	mutex.Lock()
	previous := clients
	clients = next
	mutex.Unlock()
	t.Cleanup(func() {
		mutex.Lock()
		clients = previous
		mutex.Unlock()
	})
}

func isolateClients(t *testing.T) {
	t.Helper()
	mutex.Lock()
	previous := clients
	clients = make(map[string]*memcache.Client)
	mutex.Unlock()
	t.Cleanup(func() {
		Close()
		mutex.Lock()
		clients = previous
		mutex.Unlock()
	})
}

func withMemcachedConfig(t *testing.T, config map[string]any) {
	t.Helper()
	configMap := gowok.Config.Map()
	previous, existed := configMap[plugin]
	if config == nil {
		delete(configMap, plugin)
	} else {
		configMap[plugin] = config
	}
	t.Cleanup(func() {
		if existed {
			configMap[plugin] = previous
		} else {
			delete(configMap, plugin)
		}
	})
}

func enabledConfig(endpoint string) map[string]any {
	return map[string]any{
		"enabled": true,
		"servers": []string{endpoint},
	}
}

func TestConfigFromMap(t *testing.T) {
	configs := ConfigFromMap(map[string]any{
		"default": map[string]any{"enabled": true, "servers": []string{"localhost:11211"}},
		"staging": map[string]any{"enabled": false, "servers": []string{"localhost:11212"}},
	})
	if len(configs) != 2 {
		t.Fatalf("ConfigFromMap() returned %d configs, want 2", len(configs))
	}
	if !configs["default"].Enabled || !reflect.DeepEqual(configs["default"].Servers, []string{"localhost:11211"}) {
		t.Fatalf("ConfigFromMap(default) = %+v", configs["default"])
	}
}

func TestNameOrDefault(t *testing.T) {
	if got := nameOrDefault(); got != "default" {
		t.Fatalf("nameOrDefault() = %q, want default", got)
	}
	if got := nameOrDefault("cache"); got != "cache" {
		t.Fatalf("nameOrDefault(cache) = %q, want cache", got)
	}
}

func TestConfigureOpensEnabledConnections(t *testing.T) {
	isolateClients(t)
	server := startFakeMemcached(t)
	withMemcachedConfig(t, map[string]any{
		"default":   enabledConfig(server.addr()),
		"secondary": enabledConfig(server.addr()),
		"disabled":  map[string]any{"enabled": false, "servers": []string{server.addr()}},
	})

	Configure(context.Background())

	if got, want := Names(), []string{"default", "secondary"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Names() = %v, want %v", got, want)
	}
	if err := Ping(); err != nil {
		t.Fatalf("Ping() = %v", err)
	}
}

func TestConfigureSkipsUnreachableConnections(t *testing.T) {
	isolateClients(t)
	server := startFakeMemcached(t)
	withMemcachedConfig(t, map[string]any{
		"unreachable": enabledConfig("127.0.0.1:1"),
		"valid":       enabledConfig(server.addr()),
	})

	Configure(context.Background())

	if got, want := Names(), []string{"valid"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Names() = %v, want %v", got, want)
	}
}

func TestConfigureWithoutConfiguration(t *testing.T) {
	isolateClients(t)
	withMemcachedConfig(t, nil)
	Configure(context.Background())
	if got := Names(); len(got) != 0 {
		t.Fatalf("Names() = %v, want empty", got)
	}
}

func TestConfigureClosesPreviousConnections(t *testing.T) {
	isolateClients(t)
	withMemcachedConfig(t, nil)
	stale := memcache.New("127.0.0.1:1")
	mutex.Lock()
	clients["stale"] = stale
	mutex.Unlock()

	Configure(context.Background())

	if ClientNoDefault("stale").IsPresent() {
		t.Fatal("stale connection was not removed")
	}
}

func TestConfigureRegistersCacheClient(t *testing.T) {
	isolateClients(t)
	server := startFakeMemcached(t)
	withMemcachedConfig(t, map[string]any{
		"cache-integration": enabledConfig(server.addr()),
	})
	Configure(context.Background())

	c := gowokcache.Client("cache-integration")
	if err := c.Set(context.Background(), "key", 123); err != nil {
		t.Fatalf("cache Set() = %v", err)
	}
	got, err := c.Get(context.Background(), "key")
	if err != nil {
		t.Fatalf("cache Get() = %v", err)
	}
	if got != 123 {
		t.Fatalf("cache Get() = %#v, want 123", got)
	}
}

func TestRegisterHealthReportsUpAndDown(t *testing.T) {
	isolateClients(t)
	server := startFakeMemcached(t)
	withMemcachedConfig(t, map[string]any{"default": enabledConfig(server.addr())})
	Configure(context.Background())

	previousMux := gowok.Web.HttpServeMux
	gowok.Web.HttpServeMux = ngamux.NewHttpServeMux()
	t.Cleanup(func() { gowok.Web.HttpServeMux = previousMux })
	gowok.Health.Configure()

	check := func(want string) {
		t.Helper()
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/health/memcached", nil)
		gowok.Web.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("health status = %d, want 200", response.Code)
		}
		var body map[string]string
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body["status"] != want {
			t.Fatalf("health status = %q, want %q", body["status"], want)
		}
	}

	check("UP")
	server.setError("memcached is down")
	check("DOWN")
}

func TestCloseSkipsNilClient(t *testing.T) {
	withTestClients(t, map[string]*memcache.Client{"nil": nil})
	Close()
	if got := Names(); len(got) != 0 {
		t.Fatalf("Names() = %v, want empty", got)
	}
}

func TestPingWithoutConnection(t *testing.T) {
	withTestClients(t, map[string]*memcache.Client{})
	if err := Ping(); err == nil {
		t.Fatal("Ping() without a connection should fail")
	}
}

func TestPingReportsServerError(t *testing.T) {
	server := startFakeMemcached(t)
	withTestClients(t, map[string]*memcache.Client{"default": newTestMemcacheClient(t, server)})
	server.setError("memcached is down")
	if err := Ping(); err == nil {
		t.Fatal("Ping() should fail when the server returns an error")
	}
}

func TestSetJSONAndGetJSONRoundTrip(t *testing.T) {
	server := startFakeMemcached(t)
	withTestClients(t, map[string]*memcache.Client{"default": newTestMemcacheClient(t, server)})

	in := map[string]string{"name": "gowok"}
	if err := SetJSON(context.Background(), "key", in, 0); err != nil {
		t.Fatalf("SetJSON() = %v", err)
	}
	var out map[string]string
	if err := GetJSON(context.Background(), "key", &out); err != nil {
		t.Fatalf("GetJSON() = %v", err)
	}
	if !reflect.DeepEqual(out, in) {
		t.Fatalf("GetJSON() = %v, want %v", out, in)
	}
}

func TestJSONWithoutConnection(t *testing.T) {
	withTestClients(t, map[string]*memcache.Client{})
	if err := SetJSON(context.Background(), "key", "value", 0); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("SetJSON() error = %v, want ErrNotConfigured", err)
	}
	var out any
	if err := GetJSON(context.Background(), "key", &out); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("GetJSON() error = %v, want ErrNotConfigured", err)
	}
}

func TestJSONHonorsCanceledContext(t *testing.T) {
	server := startFakeMemcached(t)
	withTestClients(t, map[string]*memcache.Client{"default": newTestMemcacheClient(t, server)})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := SetJSON(ctx, "key", "value", 0); !errors.Is(err, context.Canceled) {
		t.Fatalf("SetJSON() error = %v, want context.Canceled", err)
	}
	var out any
	if err := GetJSON(ctx, "key", &out); !errors.Is(err, context.Canceled) {
		t.Fatalf("GetJSON() error = %v, want context.Canceled", err)
	}
}

func TestSetJSONReportsMarshalError(t *testing.T) {
	server := startFakeMemcached(t)
	withTestClients(t, map[string]*memcache.Client{"default": newTestMemcacheClient(t, server)})
	if err := SetJSON(context.Background(), "key", make(chan int), 0); err == nil {
		t.Fatal("SetJSON() accepted a non-marshalable value")
	}
}

func TestGetJSONReturnsMissingKeyError(t *testing.T) {
	server := startFakeMemcached(t)
	withTestClients(t, map[string]*memcache.Client{"default": newTestMemcacheClient(t, server)})
	var out any
	if err := GetJSON(context.Background(), "missing", &out); !errors.Is(err, memcache.ErrCacheMiss) {
		t.Fatalf("GetJSON() error = %v, want memcache.ErrCacheMiss", err)
	}
}

func TestGetJSONReturnsUnmarshalError(t *testing.T) {
	server := startFakeMemcached(t)
	withTestClients(t, map[string]*memcache.Client{"default": newTestMemcacheClient(t, server)})
	if err := SetJSON(context.Background(), "key", "not-a-number", 0); err != nil {
		t.Fatal(err)
	}
	var out int
	if err := GetJSON(context.Background(), "key", &out); err == nil {
		t.Fatal("GetJSON() should fail when the stored value cannot be decoded")
	}
}

func TestSetJSONReportsServerError(t *testing.T) {
	server := startFakeMemcached(t)
	withTestClients(t, map[string]*memcache.Client{"default": newTestMemcacheClient(t, server)})
	server.setError("memcached is down")
	if err := SetJSON(context.Background(), "key", "value", 0); err == nil {
		t.Fatal("SetJSON() should fail when the server returns an error")
	}
}

func TestExpirationSeconds(t *testing.T) {
	tests := []struct {
		name string
		ttl  time.Duration
		want int32
	}{
		{name: "no expiration", ttl: 0, want: 0},
		{name: "negative", ttl: -time.Second, want: 0},
		{name: "sub second rounds up", ttl: 500 * time.Millisecond, want: 1},
		{name: "seconds round up", ttl: 1500 * time.Millisecond, want: 2},
		{name: "day", ttl: 24 * time.Hour, want: 86400},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := expirationSeconds(tt.ttl); got != tt.want {
				t.Fatalf("expirationSeconds(%v) = %d, want %d", tt.ttl, got, tt.want)
			}
		})
	}

	before := time.Now().Add(31 * 24 * time.Hour).Unix()
	got := expirationSeconds(31 * 24 * time.Hour)
	after := time.Now().Add(31 * 24 * time.Hour).Unix()
	if int64(got) < before || int64(got) > after {
		t.Fatalf("long TTL = %d, want absolute timestamp between %d and %d", got, before, after)
	}
}

func TestClientFallbackToDefault(t *testing.T) {
	server := startFakeMemcached(t)
	withTestClients(t, map[string]*memcache.Client{"default": newTestMemcacheClient(t, server)})
	if c := Client("missing"); c.IsEmpty() {
		t.Fatal("Client(missing) did not fall back to default")
	}
	if c := ClientNoDefault("missing"); !c.IsEmpty() {
		t.Fatal("ClientNoDefault(missing) returned a client")
	}
}

func TestClientNoDefaultEmpty(t *testing.T) {
	withTestClients(t, map[string]*memcache.Client{})
	if _, ok := ClientNoDefault("missing").Get(); ok {
		t.Fatal("ClientNoDefault returned an unknown client")
	}
}
