package consume

import (
	gowok_amqp "github.com/gowok/plugins/amqp"
	"github.com/gowok/plugins/amqp/channel"
	amqp "github.com/rabbitmq/amqp091-go"
)

type consuming struct {
	Channel                             *amqp.Channel
	Queue, Consumer                     string
	AutoACK, Exclusive, NoLocal, NoWait bool
	Args                                gowok_amqp.Table
}

func Consume(opts ...func(*consuming)) (<-chan amqp.Delivery, error) {
	c := consuming{}
	for _, opt := range opts {
		opt(&c)
	}

	if c.Channel == nil {
		ch, err := channel.Channel()
		if err != nil {
			return nil, err
		}
		defer ch.Close()
		c.Channel = ch
	}

	msgs, err := c.Channel.Consume(
		c.Queue,
		c.Consumer,
		c.AutoACK,
		c.Exclusive,
		c.NoLocal,
		c.NoWait,
		amqp.Table(c.Args),
	)
	if err != nil {
		return nil, err
	}

	return msgs, nil
}
func WithChannel(value *amqp.Channel) func(*consuming) {
	return func(p *consuming) {
		p.Channel = value
	}
}
func WithQueue(value string) func(*consuming) {
	return func(p *consuming) {
		p.Queue = value
	}
}
func WithConsumer(value string) func(*consuming) {
	return func(p *consuming) {
		p.Consumer = value
	}
}
func WithAutoACK(value bool) func(*consuming) {
	return func(p *consuming) {
		p.AutoACK = value
	}
}
func WithExclusive(value bool) func(*consuming) {
	return func(p *consuming) {
		p.Exclusive = value
	}
}
func WithNoLocal(value bool) func(*consuming) {
	return func(p *consuming) {
		p.NoLocal = value
	}
}
func WithNoWait(value bool) func(*consuming) {
	return func(p *consuming) {
		p.NoWait = value
	}
}
func WithArgs(value gowok_amqp.Table) func(*consuming) {
	return func(p *consuming) {
		p.Args = value
	}
}
