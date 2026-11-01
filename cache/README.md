# `cache`
It help to manage cache with several storage supports, like Redis, memory, memcache, etc.
It is high level interface that only provide general action.
Once installed and configured, it will be ready to use.

## Installation

```bash
go get github.com/gowok/plugins/cache
```

## Configuration

```go
func main() {
  gowok.Get().
    Configures(
      redis.Configure,
      cache.Configure,
    ).
    Run()
}
```

```yaml
redis:
  default:
    enabled: true
    dsn: redis://localhost:6379/0

cache:
  default:
    enabled: <bool>
    driver: <string> # "memory" or "redis"
    name: <string>   # redis connection to use, defaults to the cache entry name
    dsn: <string>    # placeholder for future use
```

## Health Check
```
http://<host>/health/cache
```
