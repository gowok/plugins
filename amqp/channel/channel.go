package channel

import (
	gowok_amqp "github.com/gowok/plugins/amqp"
	amqp "github.com/rabbitmq/amqp091-go"
)

func Channel() (*amqp.Channel, error) {
	ch, err := gowok_amqp.Connection().Channel()
	if err != nil {
		return nil, err
	}

	return ch, nil
}

func ChannelDo(callback func(*amqp.Channel) error) error {
	ch, err := gowok_amqp.Connection().Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	return callback(ch)
}
