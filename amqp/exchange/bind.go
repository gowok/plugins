package exchange

import (
	gowok_amqp "github.com/gowok/plugins/amqp"
	"github.com/gowok/plugins/amqp/channel"
	amqp "github.com/rabbitmq/amqp091-go"
)

type binding struct {
	Channel                    *amqp.Channel
	RoutingKey, Source, Target string
	NoWait                     bool
	Args                       gowok_amqp.Table
}

func Bind(source, destination, routingKey string, opts ...func(*binding)) error {
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

	return b.Channel.ExchangeBind(b.Target, b.RoutingKey, b.Source, b.NoWait, amqp.Table(b.Args))
}

func BindWithChannel(value *amqp.Channel) func(*binding) {
	return func(b *binding) {
		b.Channel = value
	}
}

func BindWithNoWait(value bool) func(*binding) {
	return func(b *binding) {
		b.NoWait = value
	}
}

func BindWithArgs(value gowok_amqp.Table) func(*binding) {
	return func(b *binding) {
		b.Args = value
	}
}
