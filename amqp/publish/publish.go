package amqp

import (
	"context"

	"github.com/gowok/plugins/amqp/channel"
	amqp "github.com/rabbitmq/amqp091-go"
)

type publishing struct {
	Context              context.Context
	Channel              *amqp.Channel
	Exchange, RoutingKey string
	Mandatory, Immediate bool
}

func Publish(message amqp.Publishing, opts ...func(*publishing)) (*amqp.Publishing, error) {
	p := publishing{}
	for _, opt := range opts {
		opt(&p)
	}

	if p.Channel == nil {
		ch, err := channel.Channel()
		if err != nil {
			return nil, err
		}
		defer ch.Close()
		p.Channel = ch
	}

	if p.Context == nil {
		p.Context = context.Background()
	}
	err := p.Channel.PublishWithContext(
		p.Context,
		p.Exchange,
		p.RoutingKey,
		p.Mandatory,
		p.Immediate,
		message,
	)
	if err != nil {
		return nil, err
	}

	return &message, nil
}
func WithContext(value context.Context) func(*publishing) {
	return func(p *publishing) {
		p.Context = value
	}
}
func WithChannel(value *amqp.Channel) func(*publishing) {
	return func(p *publishing) {
		p.Channel = value
	}
}
func WithExchange(value string) func(*publishing) {
	return func(p *publishing) {
		p.Exchange = value
	}
}
func WithRoutingKey(value string) func(*publishing) {
	return func(p *publishing) {
		p.RoutingKey = value
	}
}
func WithMandatory(value bool) func(*publishing) {
	return func(p *publishing) {
		p.Mandatory = value
	}
}
func WithImmediate(value bool) func(*publishing) {
	return func(p *publishing) {
		p.Immediate = value
	}
}
