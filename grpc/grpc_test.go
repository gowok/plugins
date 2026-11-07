package grpc

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/test/bufconn"
)

func TestConfigureWithoutConfigurationDoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Configure panicked without grpc configuration: %v", r)
		}
	}()

	Configure()()
}

func TestReflectionAlwaysRegistered(t *testing.T) {
	info := g().GetServiceInfo()

	if _, ok := info["grpc.reflection.v1.ServerReflection"]; !ok {
		t.Fatal("grpc.reflection.v1.ServerReflection not registered")
	}

	if _, ok := info["grpc.reflection.v1alpha.ServerReflection"]; !ok {
		t.Fatal("grpc.reflection.v1alpha.ServerReflection not registered")
	}
}

func TestHealthServiceRegistered(t *testing.T) {
	info := g().GetServiceInfo()

	if _, ok := info["grpc.health.v1.Health"]; !ok {
		t.Fatal("grpc.health.v1.Health not registered")
	}
}

func TestShutdownDoesNotBlockWhenNotServing(t *testing.T) {
	done := make(chan struct{})

	go func() {
		Shutdown()
		Shutdown()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Shutdown blocked on a server that is not serving")
	}
}

func TestServingStatusHelpers(t *testing.T) {
	srv := health.NewServer()
	_health(srv)
	t.Cleanup(func() { _health(health.NewServer()) })

	lis := bufconn.Listen(1024 * 1024)
	gs := grpc.NewServer()
	healthpb.RegisterHealthServer(gs, srv)

	go func() { _ = gs.Serve(lis) }()
	t.Cleanup(gs.Stop)

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("connect to health server: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	client := healthpb.NewHealthClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	statusOf := func(service string) healthpb.HealthCheckResponse_ServingStatus {
		t.Helper()

		resp, err := client.Check(ctx, &healthpb.HealthCheckRequest{Service: service})
		if err != nil {
			t.Fatalf("check %q: %v", service, err)
		}

		return resp.Status
	}

	if got := statusOf(""); got != healthpb.HealthCheckResponse_SERVING {
		t.Fatalf("overall status = %v, want SERVING by default", got)
	}

	SetNotServing()
	if got := statusOf(""); got != healthpb.HealthCheckResponse_NOT_SERVING {
		t.Fatalf("overall status = %v, want NOT_SERVING", got)
	}

	SetNotServing("checkout")
	if got := statusOf("checkout"); got != healthpb.HealthCheckResponse_NOT_SERVING {
		t.Fatalf("checkout status = %v, want NOT_SERVING", got)
	}

	SetServing("checkout")
	if got := statusOf("checkout"); got != healthpb.HealthCheckResponse_SERVING {
		t.Fatalf("checkout status = %v, want SERVING", got)
	}

	if got := statusOf(""); got != healthpb.HealthCheckResponse_NOT_SERVING {
		t.Fatalf("overall status = %v, want NOT_SERVING after a per-service update", got)
	}
}
