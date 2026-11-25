package s3

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/gowok/gowok"
	"github.com/ngamux/ngamux"
)

type healthResponse struct {
	Status  string                  `json:"status"`
	Clients map[string]healthStatus `json:"clients"`
}

type healthStatus struct {
	Status string `json:"status"`
}

func newHealthS3Server(t *testing.T) (*httptest.Server, *atomic.Bool) {
	t.Helper()

	healthy := &atomic.Bool{}
	healthy.Store(true)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/":
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<ListAllMyBucketsResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Owner><ID>test</ID><DisplayName>test</DisplayName></Owner>
  <Buckets></Buckets>
</ListAllMyBucketsResult>`))
		case r.Method == http.MethodHead && r.URL.Path == "/test-bucket":
			if !healthy.Load() {
				w.WriteHeader(http.StatusForbidden)
			}
		default:
			t.Errorf("unexpected health request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	return server, healthy
}

func requestS3Health(t *testing.T) healthResponse {
	t.Helper()

	previousMux := gowok.Web.HttpServeMux
	gowok.Web.HttpServeMux = ngamux.NewHttpServeMux()
	t.Cleanup(func() { gowok.Web.HttpServeMux = previousMux })

	gowok.Health.Configure()

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health/s3", nil)
	gowok.Web.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("health status = %d, want %d", response.Code, http.StatusOK)
	}

	var result healthResponse
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	return result
}

func TestConfigureHealthReportsUpAndDown(t *testing.T) {
	resetS3State(t)
	defaultServer, defaultHealthy := newHealthS3Server(t)
	cashierServer, cashierHealthy := newHealthS3Server(t)
	setS3Config(t, map[string]any{
		"default": enabledConfig(defaultServer.URL),
		"cashier": enabledConfig(cashierServer.URL),
	})

	Configure(nil)(context.Background())

	result := requestS3Health(t)
	if result.Status != "UP" {
		t.Fatalf("health status = %q, want UP", result.Status)
	}
	for _, name := range []string{"default", "cashier"} {
		if got := result.Clients[name].Status; got != "UP" {
			t.Fatalf("health for %q = %q, want UP", name, got)
		}
	}

	cashierHealthy.Store(false)
	result = requestS3Health(t)
	if result.Status != "DOWN" {
		t.Fatalf("health status = %q, want DOWN", result.Status)
	}
	if got := result.Clients["default"].Status; got != "UP" {
		t.Fatalf("health for default = %q, want UP", got)
	}
	if got := result.Clients["cashier"].Status; got != "DOWN" {
		t.Fatalf("health for cashier = %q, want DOWN", got)
	}

	defaultHealthy.Store(false)
	result = requestS3Health(t)
	if result.Status != "DOWN" {
		t.Fatalf("health status = %q, want DOWN", result.Status)
	}
}

func TestConfigureHealthWithoutClientsReportsDown(t *testing.T) {
	resetS3State(t)
	setS3Config(t, map[string]any{})

	Configure(nil)(context.Background())

	result := requestS3Health(t)
	if result.Status != "DOWN" {
		t.Fatalf("health status = %q, want DOWN", result.Status)
	}
	if len(result.Clients) != 0 {
		t.Fatalf("health clients = %v, want none", result.Clients)
	}
}

func TestConfigureHealthHandlesInvalidClientEntry(t *testing.T) {
	resetS3State(t)
	setS3Config(t, map[string]any{})

	Configure(nil)(context.Background())
	clients.Store(42, "invalid")
	clients.Store("invalid", "invalid")

	result := requestS3Health(t)
	if result.Status != "DOWN" {
		t.Fatalf("health status = %q, want DOWN", result.Status)
	}
	if got := result.Clients["invalid"].Status; got != "DOWN" {
		t.Fatalf("health for invalid client = %q, want DOWN", got)
	}
}
