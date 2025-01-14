package opentelemetry

import "github.com/gowok/gowok/maps"

type Config struct {
	Enabled        bool
	LocalExporter  bool
	JaegerExporter bool
	Jaeger         Jaeger
	PromotheusUrl  string
}

type Jaeger struct {
	Endpoint    string
	ServiceName string
}

func ConfigFromMap(configMap map[string]any) Config {
	c := Config{}
	_ = maps.MapToStruct(configMap, &c)
	return c
}
