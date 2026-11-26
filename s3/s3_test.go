package s3

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gowok/gowok"
)

func resetS3State(t *testing.T) {
	t.Helper()

	clients = sync.Map{}
	configMap := gowok.Config.Map()
	previous, existed := configMap["s3"]
	t.Cleanup(func() {
		clients = sync.Map{}
		if existed {
			configMap["s3"] = previous
		} else {
			delete(configMap, "s3")
		}
	})

	t.Setenv("AWS_ACCESS_KEY_ID", "test-access-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret-key")
	t.Setenv("AWS_SESSION_TOKEN", "")
	t.Setenv("AWS_PROFILE", "default")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	configPath := filepath.Join(t.TempDir(), "config")
	credentialsPath := filepath.Join(t.TempDir(), "credentials")
	if err := os.WriteFile(configPath, []byte("[default]\nregion = us-east-1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(credentialsPath, []byte("[default]\naws_access_key_id = test-access-key\naws_secret_access_key = test-secret-key\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AWS_CONFIG_FILE", configPath)
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", credentialsPath)
}

func setS3Config(t *testing.T, value any) {
	t.Helper()
	gowok.Config.Map()["s3"] = value
}

func newS3Server(t *testing.T, status int) (*httptest.Server, *atomic.Int32) {
	t.Helper()

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Method != http.MethodGet {
			t.Errorf("request method = %s, want GET", r.Method)
		}
		if got, want := r.URL.Path, "/test-bucket"; got != want {
			t.Errorf("request path = %q, want %q", got, want)
		}
		if got := r.URL.Query().Get("list-type"); got != "2" {
			t.Errorf("list-type = %q, want 2", got)
		}
		if r.Header.Get("Authorization") == "" {
			t.Error("request is missing AWS authorization header")
		}

		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(status)
		if status == http.StatusOK {
			_, _ = fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>
<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Name>test-bucket</Name>
  <Prefix></Prefix>
  <KeyCount>0</KeyCount>
  <MaxKeys>1000</MaxKeys>
  <IsTruncated>false</IsTruncated>
</ListBucketResult>`)
			return
		}
		_, _ = fmt.Fprint(w, `<Error><Code>AccessDenied</Code><Message>denied</Message></Error>`)
	}))
	t.Cleanup(server.Close)
	return server, &requests
}

func enabledConfig(endpoint string) map[string]any {
	return map[string]any{
		"enabled":           true,
		"region":            "us-east-1",
		"endpoint":          endpoint,
		"access_key_id":     "test-access-key",
		"secret_access_key": "test-secret-key",
		"bucket":            "test-bucket",
	}
}

func TestClient(t *testing.T) {
	resetS3State(t)

	first := &client{}
	second := &client{}

	tests := []struct {
		name     string
		stored   map[string]any
		names    []string
		want     *client
		wantNone bool
	}{
		{name: "empty map", wantNone: true},
		{name: "default", stored: map[string]any{"default": first}, want: first},
		{name: "named", stored: map[string]any{"worker": second}, names: []string{"worker"}, want: second},
		{name: "fallback name", stored: map[string]any{"worker": second}, names: []string{"missing", "worker"}, want: second},
		{name: "missing name", stored: map[string]any{"worker": second}, names: []string{"missing"}, wantNone: true},
		{name: "wrong type", stored: map[string]any{"default": "not a client"}, wantNone: true},
		{name: "wrong SDK client type", stored: map[string]any{"default": &s3.Client{}}, wantNone: true},
		{name: "nil client", stored: map[string]any{"default": (*client)(nil)}, wantNone: true},
		{name: "nil then valid fallback", stored: map[string]any{"default": nil, "worker": second}, names: []string{"default", "worker"}, want: second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clients = sync.Map{}
			for key, value := range tt.stored {
				clients.Store(key, value)
			}

			got := Client(tt.names...)
			if tt.wantNone {
				if !got.IsEmpty() {
					t.Fatal("Client() returned a client, want empty")
				}
				return
			}
			if got.IsEmpty() {
				t.Fatal("Client() returned empty, want a client")
			}
			client, ok := got.Get()
			if !ok || client != tt.want {
				t.Fatalf("Client() = %p, want %p", client, tt.want)
			}
		})
	}
}

func TestConfigureEmpty(t *testing.T) {
	resetS3State(t)
	setS3Config(t, map[string]any{})

	Configure(nil)(context.Background())

	if !Client().IsEmpty() {
		t.Fatal("Configure() registered a client from an empty config")
	}
}

func TestConfigureRegistersMultipleClients(t *testing.T) {
	resetS3State(t)
	server, requests := newS3Server(t, http.StatusOK)
	setS3Config(t, map[string]any{
		"default": enabledConfig(server.URL),
		"cashier": enabledConfig(server.URL),
		"staging": map[string]any{"enabled": false, "region": "us-west-2"},
	})

	Configure(nil)(context.Background())

	for _, name := range []string{"default", "cashier"} {
		client := Client(name)
		if client.IsEmpty() {
			t.Fatalf("Client(%q) returned empty, want registered client", name)
		}
	}
	if !Client("staging").IsEmpty() {
		t.Fatal("Configure() registered a disabled client")
	}
	if got := requests.Load(); got != 2 {
		t.Fatalf("S3 requests = %d, want 2", got)
	}
}

func TestConfigureSkipsInvalidEntry(t *testing.T) {
	resetS3State(t)
	server, requests := newS3Server(t, http.StatusOK)
	setS3Config(t, map[string]any{
		"invalid": map[string]any{"enabled": make(chan int)},
		"valid":   enabledConfig(server.URL),
	})

	Configure(nil)(context.Background())

	if Client("valid").IsEmpty() {
		t.Fatal("Configure() did not continue after an invalid entry")
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("S3 requests = %d, want 1", got)
	}
}

func TestConfigureSkipsInvalidEnabledType(t *testing.T) {
	resetS3State(t)
	server, requests := newS3Server(t, http.StatusOK)
	setS3Config(t, map[string]any{
		"invalid": map[string]any{"enabled": "yes"},
		"valid":   enabledConfig(server.URL),
	})

	Configure(nil)(context.Background())

	if Client("valid").IsEmpty() {
		t.Fatal("Configure() did not continue after an invalid enabled value")
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("S3 requests = %d, want 1", got)
	}
}

func TestConfigureConnectionFailure(t *testing.T) {
	resetS3State(t)
	server, requests := newS3Server(t, http.StatusForbidden)
	setS3Config(t, map[string]any{"default": enabledConfig(server.URL)})

	var gotErr error
	Configure(&ConfigureInput{OnError: func(err error) { gotErr = err }})(context.Background())

	if gotErr == nil {
		t.Fatal("Configure() did not report connection failure")
	}
	if !Client().IsEmpty() {
		t.Fatal("Configure() registered a client after connection failure")
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("S3 requests = %d, want 1", got)
	}
}

func TestConfigureDefaultErrorHandler(t *testing.T) {
	resetS3State(t)
	server, _ := newS3Server(t, http.StatusForbidden)
	setS3Config(t, map[string]any{"default": enabledConfig(server.URL)})

	// A nil input exercises both the default ConfigureInput and default OnError.
	Configure(nil)(context.Background())

	if !Client().IsEmpty() {
		t.Fatal("Configure() registered a client after connection failure")
	}
}

func TestConfigureWithoutEndpoint(t *testing.T) {
	resetS3State(t)
	config := enabledConfig("")
	delete(config, "endpoint")
	setS3Config(t, map[string]any{"default": config})

	// Cancel before the network call so this covers the empty-endpoint branch
	// without contacting AWS.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var gotErr error
	Configure(&ConfigureInput{OnError: func(err error) { gotErr = err }})(ctx)

	if gotErr == nil {
		t.Fatal("Configure() did not report the canceled request")
	}
	if !Client().IsEmpty() {
		t.Fatal("Configure() registered a client after a canceled request")
	}
}

func TestConfigureLoadConfigFailure(t *testing.T) {
	resetS3State(t)
	configPath := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(configPath, []byte("[default]\nregion = us-east-1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AWS_CONFIG_FILE", configPath)
	t.Setenv("AWS_PROFILE", "missing-profile")
	setS3Config(t, map[string]any{
		"default": map[string]any{"enabled": true, "region": "us-east-1"},
	})

	var gotErr error
	Configure(&ConfigureInput{OnError: func(err error) { gotErr = err }})(context.Background())

	if gotErr == nil {
		t.Fatal("Configure() did not report AWS config loading failure")
	}
	if !Client().IsEmpty() {
		t.Fatal("Configure() registered a client after AWS config loading failure")
	}
}

func TestConfigureHonorsCanceledContext(t *testing.T) {
	resetS3State(t)
	server, _ := newS3Server(t, http.StatusOK)
	setS3Config(t, map[string]any{"default": enabledConfig(server.URL)})

	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	var gotErr error
	Configure(&ConfigureInput{OnError: func(err error) { gotErr = err }})(ctx)

	if gotErr == nil {
		t.Fatal("Configure() did not report context cancellation")
	}
	if !Client().IsEmpty() {
		t.Fatal("Configure() registered a client after context cancellation")
	}
}

func TestConfigurePanicsOnNonMapEntry(t *testing.T) {
	resetS3State(t)
	setS3Config(t, map[string]any{"default": "not a map"})

	defer func() {
		if recover() == nil {
			t.Fatal("Configure() did not panic for a non-map entry")
		}
	}()
	Configure(nil)(context.Background())
}
