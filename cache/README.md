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
    Configures(cache.Configure).
    Run()
}
```

```yaml
cache:
  default:
    enabled: <bool>
    driver: <string>
    dsn: <string>
```

## Health Check
```
http://<host>/health/cache
```
