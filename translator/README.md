# `translator`
It help to manage translations.

## Installation

```bash
go get github.com/gowok/plugins/translator \
  github.com/go-playground/locales/en \
  github.com/go-playground/locales/id
```

## Configuration
```go
func main() {
  gowok.Get().
    Configures(
      translator.Configure(en.New, id.New),
    ).
    Run()
}
```

## Add to Dictionary
```go
translator.Add(key, text string, override bool)

// ...

translator.Add("successAdd", "insert a new {0} successfully", override)
```

## Translating
```go
translator.T(key string, params ...string)

// ...

translator.T("successAdd", "user")
// output: insert a new user successfully
```
