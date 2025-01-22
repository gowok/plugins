# `amqp`
It help to manage connection to AMQP server like RabbitMQ.
Once installed and configured, it will be ready to use.
Also, it provides several utilities to shorten the process, like creating channel, queue, exchange, etc.

## Installation

```bash
go get github.com/gowok/plugins/amqp
```

## Configuration

```go
func main() {
  gowok.Get().
    Configures(amqp.Configure).
    Run()
}
```

```yaml
amqp:
  enabled: <bool>
  dsn: <string>
```

## Health Check
```
http://<host>/health/amqp
```
