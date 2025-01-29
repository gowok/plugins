package channel

import (
	gowok_amqp "github.com/gowok/plugins/amqp"
	amqp "github.com/rabbitmq/amqp091-go"
)

func New() (*amqp.Channel, error) {
	c := gowok_amqp.Connection().OrPanic()
	ch, err := c.Channel()
	if err != nil {
		return nil, err
	}

	return ch, nil
}
