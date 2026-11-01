package queue

import (
	"github.com/gowok/plugins/amqp/channel"
	amqp "github.com/rabbitmq/amqp091-go"
)

type deleting struct {
	Channel           *amqp.Channel
	IfUnused, IfEmpty bool
	NoWait            bool
}

func Delete(name string, opts ...func(*deleting)) error {
	d := deleting{}
	for _, opt := range opts {
		if opt != nil {
			opt(&d)
		}
	}

	if d.Channel == nil {
		ch, err := channel.New()
		if err != nil {
			return err
		}
		defer ch.Close()
		d.Channel = ch
	}

	_, err := d.Channel.QueueDelete(name, d.IfUnused, d.IfEmpty, d.NoWait)
	return err
}

func DeleteWithChannel(value *amqp.Channel) func(*deleting) {
	return func(d *deleting) {
		d.Channel = value
	}
}

func DeleteIfUnused(value bool) func(*deleting) {
	return func(d *deleting) {
		d.IfUnused = value
	}
}

func DeleteIfEmpty(value bool) func(*deleting) {
	return func(d *deleting) {
		d.IfEmpty = value
	}
}

func DeleteNoWait(value bool) func(*deleting) {
	return func(d *deleting) {
		d.NoWait = value
	}
}
