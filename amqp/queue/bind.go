package queue

import (
	gowok_amqp "github.com/gowok/plugins/amqp"
	"github.com/gowok/plugins/amqp/channel"
	amqp "github.com/rabbitmq/amqp091-go"
)

type bind struct {
	Name, RoutingKey, Exchange string
	NoWait                     bool
	Channel                    *amqp.Channel
	Args                       gowok_amqp.Table
}

func Bind(queue, exchange string, opts ...func(*bind)) error {
	q := bind{Name: queue, Exchange: exchange}
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

	err := q.Channel.QueueBind(
		q.Name,
		q.RoutingKey,
		q.Exchange,
		q.NoWait,
		amqp.Table(q.Args),
	)
	if err != nil {
		return err
	}

	return nil
}

func BindWithChannel(value *amqp.Channel) func(*bind) {
	return func(q *bind) {
		q.Channel = value
	}
}
func BindWithRoutingKey(value string) func(*bind) {
	return func(q *bind) {
		q.RoutingKey = value
	}
}
func BindWithNoWait(value bool) func(*bind) {
	return func(q *bind) {
		q.NoWait = value
	}
}
func BindWithArgs(value gowok_amqp.Table) func(*bind) {
	return func(q *bind) {
		q.Args = value
	}
}
