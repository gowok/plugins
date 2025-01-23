# `gorm`
It help to manage database connection through GORM.

## Installation

```bash
go get github.com/gowok/plugins/gorm
```

## Configuration

```go
func main() {
  gowok.Get().
    Configures(
      gorm.Configure(map[string]gorm.Opener{
        "<driver>": driver.Open,
      }),
    ).
    Run()
}
```

```yaml
gorm:
  default:
    enabled: <bool>
    driver: <string>
    dsn: <string>
```

Please note that `driver` inside YAML config should match with `driver` inside main function.

## Health Check
```
http://<host>/health/gorm
```
