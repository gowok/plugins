package amqp

import (
	"fmt"
	"log/slog"

	"math/rand"

	"github.com/gowok/fp/maps"
	"github.com/gowok/gowok"
	"github.com/gowok/gowok/singleton"
	"github.com/gowok/gowok/some"
	"github.com/ngamux/ngamux"

	amqp "github.com/rabbitmq/amqp091-go"
)

var plugin = "amqp"

var connection = singleton.New(func() *amqp.Connection {
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

func Configure(onError func(error)) func() {
	return func() {
		slog := slog.With("plugin", plugin)
		if onError == nil {
			onError = func(err error) {}
		}

		configMap := maps.Get(gowok.Config.Map(), "amqp", map[string]any{})
		var config Config
		err := maps.ToStruct(configMap, &config)
		if err != nil {
			onError(err)
			slog.Warn("failed to map configuration", "error", err)
			return
		}
		if !config.Enabled {
			return
		}

		c, err := amqp.Dial(config.DSN)
		if err != nil {
			onError(err)
			slog.Warn("failed to connect", "error", err)
			return
		}

		gowok.Health.Add("amqp", healthFunc(c))

		closeChan := make(chan *amqp.Error)
		c.NotifyClose(closeChan)

		go func() {
			err = <-closeChan
			if err != nil {
				onError(err)
				slog.Warn("connection closed, reconnecting", "error", err)
			}

			c.Close()
		}()

		connection(c)
	}
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

func healthFunc(c *amqp.Connection) func() any {
	return func() any {
		status := ngamux.Map{"status": "DOWN"}
		ch, err := c.Channel()
		if err != nil {
			return status
		}
		q, err := ch.QueueDeclare(
			"",
			false,
			false,
			true,
			false,
			nil,
		)
		if err != nil {
			return status
		}
		defer func() {
			ch.QueueDelete(q.Name, false, false, true)
			ch.Close()
		}()
		body := fmt.Sprintf("%d", rand.Intn(100))
		err = ch.Publish(
			"",
			q.Name,
			false,
			false,
			amqp.Publishing{
				ContentType: "text/plain",
				Body:        []byte(body),
			})
		if err != nil {
			return status
		}
		msgs, err := ch.Consume(
			q.Name,
			"",
			true,
			true,
			false,
			false,
			nil,
		)
		if err != nil {
			return status
		}

		d := <-msgs
		if string(d.Body) != body {
			return status
		}

		return ngamux.Map{"status": "UP"}
	}
}
