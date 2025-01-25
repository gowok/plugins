# `mongo`
It help to manage MongoDB connection(s).

## Installation

```bash
go get github.com/gowok/plugins/mongo
```

## Configuration

```go
func main() {
  gowok.Get().
    Configures(
      mongo.Configure,
    ).
    Run()
}
```

```yaml
mongo:
  default:
    enabled: <bool>
    dsn: <string>
```

## Health Check
```
http://<host>/health/mongo
```
