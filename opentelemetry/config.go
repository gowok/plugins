package opentelemetry

import "github.com/gowok/gowok/maps"

type Config struct {
	Enabled        bool
	ServiceName    string
	LocalExporter  bool
	JaegerExporter Jaeger
	MetricExporter MetricExporter
}

type Jaeger struct {
	Enabled  bool
	Endpoint string
}

type MetricExporter struct {
	Enabled bool
	Path    string
}

func ConfigFromMap(configMap map[string]any) Config {
	c := Config{}
	_ = maps.MapToStruct(configMap, &c)
	return c
}
