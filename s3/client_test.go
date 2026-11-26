package s3

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"github.com/gowok/gowok/storage"
)

var _ storage.Storage = client{}

func newTestClient(t *testing.T, endpoint string, httpClient aws.HTTPClient) client {
	t.Helper()

	cfg := aws.Config{
		Region:       "us-east-1",
		Credentials:  aws.AnonymousCredentials{},
		HTTPClient:   httpClient,
		BaseEndpoint: aws.String(endpoint),
	}
	sdkClient := awss3.NewFromConfig(cfg, func(options *awss3.Options) {
		options.UsePathStyle = true
	})
	return client{
		Client: *sdkClient,
		config: Config{Bucket: "test-bucket"},
	}
}

func TestClientConfig(t *testing.T) {
	want := Config{
		Enabled:         true,
		Region:          "us-east-1",
		Endpoint:        "http://localhost:9000",
		AccessKeyID:     "test-access-key",
		SecretAccessKey: "test-secret-key",
		Bucket:          "test-bucket",
		ForcePathStyle:  true,
		Timeout:         5 * time.Second,
	}
	c := client{config: want}

	if got := c.Config(); got != want {
		t.Fatalf("Config() = %+v, want %+v", got, want)
	}
}

func TestClientGet(t *testing.T) {
	const key = "folder/object.txt"
	const content = "object contents"

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if got, want := r.URL.Path, "/test-bucket/"+key; got != want {
			t.Errorf("path = %q, want %q", got, want)
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = io.WriteString(w, content)
	}))
	defer server.Close()

	got, err := newTestClient(t, server.URL, server.Client()).Get(context.Background(), key)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if string(got) != content {
		t.Fatalf("Get() = %q, want %q", got, content)
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("requests = %d, want 1", got)
	}
}

func TestClientGetObjectError(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `<Error><Code>NoSuchKey</Code><Message>missing</Message></Error>`)
	}))
	defer server.Close()

	got, err := newTestClient(t, server.URL, server.Client()).Get(context.Background(), "missing")
	if err == nil {
		t.Fatal("Get() error = nil, want an error")
	}
	if got != nil {
		t.Fatalf("Get() = %q, want nil", got)
	}
	var apiErr smithy.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("Get() error = %T, want smithy.APIError", err)
	}
	if got, want := apiErr.ErrorCode(), "NoSuchKey"; got != want {
		t.Fatalf("error code = %q, want %q", got, want)
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("requests = %d, want 1", got)
	}
}

func TestClientGetReadErrorClosesBody(t *testing.T) {
	readErr := errors.New("read failed")
	body := &trackingErrorBody{err: readErr}
	var requests atomic.Int32

	httpClient := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requests.Add(1)
		return &http.Response{
			StatusCode:    http.StatusOK,
			Header:        make(http.Header),
			Body:          body,
			ContentLength: -1,
			Request:       r,
		}, nil
	})

	got, err := newTestClient(t, "https://s3.example.test", httpClient).Get(context.Background(), "key")
	if !errors.Is(err, readErr) {
		t.Fatalf("Get() error = %v, want %v", err, readErr)
	}
	if got != nil {
		t.Fatalf("Get() = %q, want nil", got)
	}
	if !body.closed {
		t.Fatal("Get() did not close the response body after a read error")
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("requests = %d, want 1", got)
	}
}

func TestClientGetSuccessClosesBody(t *testing.T) {
	body := &trackingBody{Reader: strings.NewReader("contents")}
	httpClient := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode:    http.StatusOK,
			Header:        make(http.Header),
			Body:          body,
			ContentLength: -1,
			Request:       r,
		}, nil
	})

	got, err := newTestClient(t, "https://s3.example.test", httpClient).Get(context.Background(), "key")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if string(got) != "contents" {
		t.Fatalf("Get() = %q, want %q", got, "contents")
	}
	if !body.closed {
		t.Fatal("Get() did not close the response body")
	}
}

func TestClientPut(t *testing.T) {
	tests := []struct {
		name string
		body []byte
	}{
		{name: "text", body: []byte("contents")},
		{name: "binary", body: []byte{0x00, 0x01, 0xfe, 0xff}},
		{name: "empty", body: []byte{}},
		{name: "nil", body: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if r.Method != http.MethodPut {
					t.Errorf("method = %s, want PUT", r.Method)
				}
				if got, want := r.URL.Path, "/test-bucket/key"; got != want {
					t.Errorf("path = %q, want %q", got, want)
				}
				got, err := io.ReadAll(r.Body)
				if err != nil {
					t.Errorf("read request body: %v", err)
				}
				if !bytes.Equal(got, tt.body) {
					t.Errorf("body = %v, want %v", got, tt.body)
				}
				w.Header().Set("ETag", `"test-etag"`)
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			err := newTestClient(t, server.URL, server.Client()).Put(context.Background(), "key", tt.body)
			if err != nil {
				t.Fatalf("Put() error = %v", err)
			}
			if got := requests.Load(); got != 1 {
				t.Fatalf("requests = %d, want 1", got)
			}
		})
	}
}

func TestClientPutObjectError(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `<Error><Code>AccessDenied</Code><Message>denied</Message></Error>`)
	}))
	defer server.Close()

	err := newTestClient(t, server.URL, server.Client()).Put(context.Background(), "key", []byte("contents"))
	if err == nil {
		t.Fatal("Put() error = nil, want an error")
	}
	var apiErr smithy.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("Put() error = %T, want smithy.APIError", err)
	}
	if got, want := apiErr.ErrorCode(), "AccessDenied"; got != want {
		t.Fatalf("error code = %q, want %q", got, want)
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("requests = %d, want 1", got)
	}
}

func TestClientHonorsCanceledContext(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := newTestClient(t, server.URL, server.Client())

	if _, err := c.Get(ctx, "key"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Get() error = %v, want context.Canceled", err)
	}
	if err := c.Put(ctx, "key", []byte("contents")); !errors.Is(err, context.Canceled) {
		t.Fatalf("Put() error = %v, want context.Canceled", err)
	}
	if got := requests.Load(); got != 0 {
		t.Fatalf("requests = %d, want 0", got)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) Do(r *http.Request) (*http.Response, error) {
	return f(r)
}

type trackingBody struct {
	io.Reader
	closed bool
}

func (b *trackingBody) Close() error {
	b.closed = true
	return nil
}

type trackingErrorBody struct {
	err    error
	closed bool
}

func (b *trackingErrorBody) Read([]byte) (int, error) {
	return 0, b.err
}

func (b *trackingErrorBody) Close() error {
	b.closed = true
	return nil
}
