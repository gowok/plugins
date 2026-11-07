package grpc

import (
	"testing"
	"time"
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
