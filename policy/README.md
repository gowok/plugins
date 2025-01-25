# `policy`
It help to manage authorization rules.
Behind the scene, it uses [Casbin](https://casbin.org).

## Installation

```bash
go get github.com/gowok/plugins/policy
```

## Configuration
```go
func main() {
  gowok.Get().
    Configures(
      policy.Configure(model),
    ).
    Run()
}
```

Available models are `rbac`, `abac`, and you can [write yourself](https://casbin.org/docs/category/model).

```yaml
sql:
  policy:
    enabled: <bool>
    driver: <string>
    dsn: <string>

# or
cache:
  policy:
    enabled: <bool>
    driver: <string>
    dsn: <string>
```

You can use SQL or cache driver as policy storage adaptor, just give it connection name `policy`.

## Enforcer
You can play with authorization rules using enforcer.
To access it, you should get it from policy package.

```go
policy.Enforcer()
```
