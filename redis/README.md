# `redis`
It help to manage Redis connection(s).

## Installation

```bash
go get github.com/gowok/plugins/redis
```

## Configuration

```go
func main() {
  gowok.Get().
    Configures(
      redis.Configure,
    ).
    Run()
}
```

```yaml
redis:
  default:
    enabled: <bool>
    dsn: <string>
```

Every enabled connection is validated and pinged while configuring, so an
unreachable server fails loudly at startup instead of on first use.

## Usage

```go
client, ok := redis.Client().Get()
if !ok {
  // no connection configured
}
```

A name that is not configured falls back to `default`:

```go
redis.Client("cache")
```

`redis.ClientNoDefault` skips that fallback, `redis.Names` lists what is open,
and `redis.Ping` checks a single connection.

## Health Check
```
http://<host>/health/redis
```

Each additional connection is exposed as `http://<host>/health/redis-<name>`.

## Shutdown

```go
redis.Close()
```
