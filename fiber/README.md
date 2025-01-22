# `fiber`
It help to build HTTP server using [Fiber](https://gofiber.io) web framework.
Once installed, it need to be configured, and optionally disable internal Gowok HTTP server.

## Installation

```bash
go get github.com/gowok/plugins/fiber
```

## Configuration

```go
import "github.com/gowok/plugins/fiber/router"

func main() {
  project := gowok.Get()
  project.Configures(router.Configure)

  router.Get("/", func(c *fiber.Ctx) error {
    return c.JSON(fiber.Map{"status": "asek"})
  })

  project.Run()
}
```

```yaml
fiber:
  enabled: <bool>
  host: <string>
```
