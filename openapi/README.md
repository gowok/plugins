# `openapi`
It help to write API documentation and serve it if you want.

## Installation

```bash
go get github.com/gowok/plugins/openapi
```

## Configuration

```go
func main() {
  gowok.Get().
    Configures(apiDocs).
    Run()
}

func apiDocs(project *gowok.Project) {
  router.Get("/docs", openapi.Docs().ServeHTTP)
}
```

```yaml
openapi: <path to openapi spec>
```
Or,
```yaml
openapi:
  title: your title
  description: your description
  version: v.what.you.want.1
  host: your.host.com
  terms_of_service: don't know just provide it
  schemes: [https, http, ws]
  security_definitions:
    bearer:
      type: apiKey
      field_name: Authorization
      value_source: header
```

## Add New Docs Record
Add to OpenAPI spec in YAML.

Or, in your code.
```go
type route struct {
	method, path string
}

func (r route) Method() string {
	return r.method
}
func (r route) Path() string {
	return r.path
}

openapi.Docs().Add("get all products", func(operation *spec.Operation) {
  // TODO: use operation object to play with this record
})(route{"GET", "/products"})

openapi.Docs().Add("create new product", func(operation *spec.Operation) {
  // TODO: use operation object to play with this record
})(route{"POST", "/products"})
```

## Access
It will accessible at `/docs`.
