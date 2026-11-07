package mongo

import (
	"context"
	"testing"

	"github.com/gowok/gowok"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// newTestClient returns a lazy client: mongo.Connect does not dial the server,
// so the client can be disconnected again without a running database.
func newTestClient(t *testing.T) *mongo.Client {
	t.Helper()

	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://127.0.0.1:27017"))
	if err != nil {
		t.Fatalf("failed to create test client: %v", err)
	}

	return client
}

func TestCloseOnEmptyState(t *testing.T) {
	prev := mongos
	mongos = make(map[string]*mongo.Client)
	t.Cleanup(func() { mongos = prev })

	Close()
	Close()

	if len(mongos) != 0 {
		t.Fatalf("expected no connections, got %d", len(mongos))
	}
	if Client().IsPresent() {
		t.Fatal("expected no client after Close")
	}
}

func TestCloseDisconnectsAndForgetsClients(t *testing.T) {
	prev := mongos
	mongos = map[string]*mongo.Client{
		"default": newTestClient(t),
		"broken":  nil,
	}
	t.Cleanup(func() { mongos = prev })

	Close()

	if len(mongos) != 0 {
		t.Fatalf("expected the registry to be empty, got %d", len(mongos))
	}
	if ClientNoDefault("default").IsPresent() {
		t.Fatal("closed client should no longer be reachable")
	}
	if ClientNoDefault("broken").IsPresent() {
		t.Fatal("nil client should not be reachable")
	}
}

func TestConfigureDropsConnectionsFromPreviousConfig(t *testing.T) {
	prev := mongos
	mongos = map[string]*mongo.Client{"stale": newTestClient(t)}
	t.Cleanup(func() { mongos = prev })

	config := gowok.Config.Map()
	prevConfig, hadConfig := config["mongo"]
	config["mongo"] = map[string]any{}
	t.Cleanup(func() {
		if hadConfig {
			config["mongo"] = prevConfig
		} else {
			delete(config, "mongo")
		}
	})

	Configure()

	if ClientNoDefault("stale").IsPresent() {
		t.Fatal("connection from the previous configuration should be closed")
	}
	if len(mongos) != 0 {
		t.Fatalf("expected no connections, got %d", len(mongos))
	}
}
