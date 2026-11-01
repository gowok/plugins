package exchange

import (
	gowok_amqp "github.com/gowok/plugins/amqp"
	"github.com/gowok/plugins/amqp/channel"
	amqp "github.com/rabbitmq/amqp091-go"
)

func Unbind(source, destination, routingKey string, opts ...func(*binding)) error {
	b := binding{Source: source, Target: destination, RoutingKey: routingKey}
	for _, opt := range opts {
		if opt != nil {
			opt(&b)
		}
	}

	if b.Channel == nil {
		ch, err := channel.New()
		if err != nil {
			return err
		}
		defer ch.Close()
		b.Channel = ch
	}

	return b.Channel.ExchangeUnbind(
		b.Target,
		b.RoutingKey,
		b.Source,
		b.NoWait,
		amqp.Table(b.Args),
	)
}

func UnbindWithChannel(value *amqp.Channel) func(*binding) {
	return func(b *binding) {
		b.Channel = value
	}
}

func UnbindWithNoWait(value bool) func(*binding) {
	return func(b *binding) {
		b.NoWait = value
	}
}

func UnbindWithArgs(value gowok_amqp.Table) func(*binding) {
	return func(b *binding) {
		b.Args = value
	}
}
