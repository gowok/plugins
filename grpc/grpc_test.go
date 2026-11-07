package grpc

import "testing"

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
