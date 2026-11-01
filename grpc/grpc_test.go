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
