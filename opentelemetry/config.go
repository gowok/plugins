package opentelemetry

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
