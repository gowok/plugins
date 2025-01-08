package amqp

import (
	"context"
	"log"
	"log/slog"
	"time"

	"github.com/gowok/gowok"
	"github.com/gowok/gowok/maps"

	amqp "github.com/rabbitmq/amqp091-go"
)

var plugin = "rabbitmq"

var connection = gowok.Singleton(func() *amqp.Connection {
	return nil
})

func Connection() *amqp.Connection {
	return *connection()
}

func Configure(project *gowok.Project) {
	configAny, ok := project.ConfigMap["rabbitmq"]
	if !ok {
		slog.Warn("no configuration", "plugin", plugin)
		return
	}
	configMap, ok := configAny.(map[string]any)
	if !ok {
		slog.Warn("no configuration", "plugin", plugin)
		return
	}
	var config Config
	err := maps.MapToStruct(configMap, &config)
	if err != nil {
		slog.Warn("failed to map configuration", "plugin", plugin, "error", err)
		return
	}

	c, err := amqp.Dial(config.DSN)
	if err != nil {
		slog.Warn("failed to connect", "plugin", plugin, "error", err)
		return
	}
	connection(c)
}

func Channel() (*amqp.Channel, error) {
	ch, err := Connection().Channel()
	if err != nil {
		return nil, err
	}

	return ch, nil
}

func ChannelDo(callback func(*amqp.Channel) error) error {
	ch, err := Connection().Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	return callback(ch)
}

func main() {
	ChannelDo(func(ch *amqp.Channel) error {
		q, err := ch.QueueDeclare(
			"hello", // name
			false,   // durable
			false,   // delete when unused
			false,   // exclusive
			false,   // no-wait
			nil,     // arguments
		)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		body := "Hello World!"
		err = ch.PublishWithContext(ctx,
			"",     // exchange
			q.Name, // routing key
			false,  // mandatory
			false,  // immediate
			amqp.Publishing{
				ContentType: "text/plain",
				Body:        []byte(body),
			})
		if err != nil {
			return err
		}
		log.Printf(" [x] Sent %s\n", body)

		return nil
	})
}
