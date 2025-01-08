package amqp

import amqp "github.com/rabbitmq/amqp091-go"

type queue struct {
	Name       string
	Durable    bool
	AutoDelete bool
	Exclusive  bool
	NoWait     bool
	Arguments  Table
	Channel    *amqp.Channel
}

func Queue(name string, opts ...func(*queue)) (amqp.Queue, error) {
	q := queue{Name: name}
	for _, opt := range opts {
		opt(&q)
	}

	if q.Channel == nil {
		ch, err := Channel()
		if err != nil {
			return amqp.Queue{}, err
		}
		defer ch.Close()
		q.Channel = ch
	}

	qq, err := q.Channel.QueueDeclare(
		"hello", // name
		false,   // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
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
func WithArguments(value Table) func(*queue) {
	return func(q *queue) {
		q.Arguments = value
	}
}
