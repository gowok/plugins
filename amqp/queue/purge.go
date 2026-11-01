package queue

import (
	"github.com/gowok/plugins/amqp/channel"
	amqp "github.com/rabbitmq/amqp091-go"
)

type purging struct {
	Channel *amqp.Channel
	NoWait  bool
}

func Purge(name string, opts ...func(*purging)) (int, error) {
	p := purging{}
	for _, opt := range opts {
		if opt != nil {
			opt(&p)
		}
	}

	if p.Channel == nil {
		ch, err := channel.New()
		if err != nil {
			return 0, err
		}
		defer ch.Close()
		p.Channel = ch
	}

	return p.Channel.QueuePurge(name, p.NoWait)
}

func PurgeWithChannel(value *amqp.Channel) func(*purging) {
	return func(p *purging) {
		p.Channel = value
	}
}

func PurgeNoWait(value bool) func(*purging) {
	return func(p *purging) {
		p.NoWait = value
	}
}
