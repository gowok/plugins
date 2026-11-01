package amqp

import "time"

type Config struct {
	Enabled   bool
	DSN       string
	Heartbeat time.Duration
}
