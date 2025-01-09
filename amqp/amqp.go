package amqp

import (
	"log/slog"

	"github.com/gowok/gowok"
	"github.com/gowok/gowok/maps"
	"github.com/gowok/gowok/some"

	amqp "github.com/rabbitmq/amqp091-go"
)

var plugin = "amqp"

var connection = gowok.Singleton(func() *amqp.Connection {
	return nil
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
	var config Config
	err := maps.MapToStruct(maps.Get(project.ConfigMap, "amqp", map[string]any{}), &config)
	if err != nil {
		slog.Warn("failed to map configuration", "plugin", plugin, "error", err)
		return
	}
	if !config.Enabled {
		return
	}

	c, err := amqp.Dial(config.DSN)
	if err != nil {
		slog.Warn("failed to connect", "plugin", plugin, "error", err)
		return
	}
	connection(c)
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
