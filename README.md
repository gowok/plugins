Set of plugins that can used with in Gowok foundation library.
Each plugin is designed to work together with Gowok internal or other plugins.
You can turn off some Gowok functions and activate any plugins to replace it functionality.

Here some of available plugins:
* [amqp](https://github.com/gowok/plugins/tree/master/amqp) - Interact to AMQP protocol, usually used for utilize [RabbitMQ](https://www.rabbitmq.com).
* [cache](https://github.com/gowok/plugins/tree/master/cache) - High level cache interface with multiple storage support.
* [fiber](https://github.com/gowok/plugins/tree/master/fiber) - HTTP server library if you won't to use what Gowok has.
* [gorm](https://github.com/gowok/plugins/tree/master/gorm) - Manage multiple connection of [GORM](https://gorm.io).
* [mongo](https://github.com/gowok/plugins/tree/master/mongo) - Manage multiple connection of [MongoDB](https://www.mongodb.com)
* [openapi](https://github.com/gowok/plugins/tree/master/openapi) - Serving API documentation with [OpenAPI](https://www.openapis.org) standard.
* [opentelemetry](https://github.com/gowok/plugins/tree/master/opentelemetry) - Metrics and tracers telemetry exporter.
* [policy](https://github.com/gowok/plugins/tree/master/policy) - Manage access rules to authorize user with all [Casbin](https://casbin.org) abilities.
* [translator](https://github.com/gowok/plugins/tree/master/translator) - Utilities to make translation easier to use.
* [validator](https://github.com/gowok/plugins/tree/master/validator) - Utilities to make validation easier to use.

## Usage
In general, a plugin has public function that called `Configure` as entry point.
It receive Gowok project object as parameter, then continue to the factory.
After that, plugin is ready to use.

Overview of `Configure` function:
```go
func Configure(project *gowok.Project)
```
Example:
```go
plugin1.Configure(gowok.Get())
plugin2.Configure(gowok.Get())
plugin3.Configure(gowok.Get())
```
Or, inside Gowok `Configures` function:
```go
gowok.Get().Configures(
  plugin1.Configure,
  plugin2.Configure,
  plugin3.Configure,
)
```

After the `Configure` function called, other things are managed by plugin.
Each plugin has different behaviour based on how it will be used.
Reading the plugin documentation is highly recommended.

## Plugin Creation
After knowing what plugins are provided here and how to use them, you can imagine that creating a plugin is easy.

### `Configure`
First thing you need is `Configure` function.
Just start to make it into your plugin and continue.

### Read Configuration
Because `Configure` function receive a Gowok project object, you can read configuration on that.

If you want to read configuration managed by Gowok, do it.
```go
project.Config
```

If you want to read raw configuration in `map[string]any`, do it.
```go
project.ConfigMap
```

After that, you can use that to making your plugin.

## Contribution
You can make your own plugin.
Or, if you want to contribute in this plugin collection, feel free to make an issue or lovely pull request 😎
