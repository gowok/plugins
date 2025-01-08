package amqp

import (
	gowok_amqp "github.com/gowok/plugins/amqp"
	"github.com/gowok/plugins/amqp/channel"
	amqp "github.com/rabbitmq/amqp091-go"
)

type queue struct {
	Name       string
	Durable    bool
	AutoDelete bool
	Exclusive  bool
	NoWait     bool
	Args       gowok_amqp.Table
	Channel    *amqp.Channel
}

func Queue(name string, opts ...func(*queue)) (amqp.Queue, error) {
	q := queue{Name: name}
	for _, opt := range opts {
		opt(&q)
	}

	if q.Channel == nil {
		ch, err := channel.Channel()
		if err != nil {
			return amqp.Queue{}, err
		}
		defer ch.Close()
		q.Channel = ch
	}

	qq, err := q.Channel.QueueDeclare(
		q.Name,
		q.Durable,
		q.AutoDelete,
		q.Exclusive,
		q.NoWait,
		amqp.Table(q.Args),
	)
	if err != nil {
		return amqp.Queue{}, err
	}

	return qq, nil
}

func WithChannel(value *amqp.Channel) func(*queue) {
	return func(q *queue) {
		q.Channel = value
	}
}
func WithDurable(value bool) func(*queue) {
	return func(q *queue) {
		q.Durable = value
	}
}
func WithAutoDelete(value bool) func(*queue) {
	return func(q *queue) {
		q.AutoDelete = value
	}
}
func WithExclusive(value bool) func(*queue) {
	return func(q *queue) {
		q.Exclusive = value
	}
}
func WithNoWait(value bool) func(*queue) {
	return func(q *queue) {
		q.NoWait = value
	}
}
func WithArgs(value gowok_amqp.Table) func(*queue) {
	return func(q *queue) {
		q.Args = value
	}
}
