package amqp

import (
	"log/slog"

	"github.com/gowok/gowok"
	"github.com/gowok/gowok/maps"
	"github.com/gowok/gowok/singleton"
	"github.com/gowok/gowok/some"

	amqp "github.com/rabbitmq/amqp091-go"
)

var plugin = "amqp"

var connection = singleton.New(func() *amqp.Connection {
	configMap := maps.Get(gowok.Get().ConfigMap, "amqp", map[string]any{})
	var config Config
	err := maps.ToStruct(configMap, &config)
	if err != nil {
		slog.Warn("failed to map configuration", "plugin", plugin, "error", err)
		return nil
	}
	if !config.Enabled {
		return nil
	}

	c, err := amqp.Dial(config.DSN)
	if err != nil {
		slog.Warn("failed to connect", "plugin", plugin, "error", err)
		return nil
	}

	return c
})

func Connection() some.Some[*amqp.Connection] {
	c := connection()
	if c == nil {
		return some.Empty[*amqp.Connection]()
	}
	if *c == nil {
		return some.Empty[*amqp.Connection]()
	}

	return some.Of(*c)
}

func Configure(project *gowok.Project) {
	_ = connection()
}

type Message struct {
	Headers Table
	Tag     uint64
	Message []byte
}

type Table map[string]any

func (t Table) Validate() error {
	return nil
}
