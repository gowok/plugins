# `memcached`

Manage multiple Memcached clients from Gowok configuration.

## Installation

```bash
go get github.com/gowok/plugins/memcached
```

## Configuration

```go
func main() {
	gowok.Get().Configures(
		memcached.Configure,
	).Run()
}
```

```yaml
memcached:
  default:
    enabled: true
    servers:
      - localhost:11211

  staging:
    enabled: true
    servers:
      - cache-1.internal:11211
      - cache-2.internal:11211
```

Each enabled entry is pinged during startup.
Unreachable entries are skipped.

## Usage

Retrieve a native Memcached client:
```go
client := memcached.Client().OrPanic()
err := client.Set(&memcache.Item{
	Key:   "greeting",
	Value: []byte("hello"),
})
```

A missing name falls back to `default`.
Use `ClientNoDefault` to disable that fallback, `Names` to list configured clients, and `Ping` to check one client.

## JSON Helpers

```go
err := memcached.SetJSON(ctx, "user", user, time.Minute)
if err != nil {
	return err
}

var loaded User
err = memcached.GetJSON(ctx, "user", &loaded)
```

Expiration is converted to Memcached seconds.
TTLs longer than 30 days are sent as absolute Unix timestamps.

## Health Check

```text
http://<host>/health/memcached
```

Additional clients use `http://<host>/health/memcached-<name>`.

## Shutdown

```go
memcached.Close()
```
