package exchange

import (
	gowok_amqp "github.com/gowok/plugins/amqp"
	"github.com/gowok/plugins/amqp/channel"
	amqp "github.com/rabbitmq/amqp091-go"
)

type exchange struct {
	Name, Type                            string
	Durable, AutoDelete, Internal, NoWait bool
	Args                                  gowok_amqp.Table
	Channel                               *amqp.Channel
}

func New(name string, opts ...func(*exchange)) error {
	q := exchange{Name: name}
	for _, opt := range opts {
		opt(&q)
	}

	if q.Channel == nil {
		ch, err := channel.New()
		if err != nil {
			return err
		}
		defer ch.Close()
		q.Channel = ch
	}

	err := q.Channel.ExchangeDeclare(
		q.Name,
		q.Type,
		q.Durable,
		q.AutoDelete,
		q.Internal,
		q.NoWait,
		amqp.Table(q.Args),
	)
	if err != nil {
		return err
	}

	return nil
}

func WithChannel(value *amqp.Channel) func(*exchange) {
	return func(q *exchange) {
		q.Channel = value
	}
}
func WithType(value string) func(*exchange) {
	return func(q *exchange) {
		q.Type = value
	}
}
func WithDurable(value bool) func(*exchange) {
	return func(q *exchange) {
		q.Durable = value
	}
}
func WithAutoDelete(value bool) func(*exchange) {
	return func(q *exchange) {
		q.AutoDelete = value
	}
}
func WithInternal(value bool) func(*exchange) {
	return func(q *exchange) {
		q.Internal = value
	}
}
func WithNoWait(value bool) func(*exchange) {
	return func(q *exchange) {
		q.NoWait = value
	}
}
func WithArgs(value gowok_amqp.Table) func(*exchange) {
	return func(q *exchange) {
		q.Args = value
	}
}
