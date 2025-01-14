package opentelemetry

import "github.com/gowok/gowok/maps"

type Config struct {
	Enabled        bool
	ServiceName    string
	LocalExporter  bool
	JaegerEnabled  bool
	JaegerExporter Jaeger
	PromotheusUrl  string
	MatricEnabled  bool
	MatricExporter MetricExporter
}

type Jaeger struct {
	Endpoint string
}

type MetricExporter struct {
	Path string
}

func ConfigFromMap(configMap map[string]any) Config {
	c := Config{}
	_ = maps.MapToStruct(configMap, &c)
	return c
}
