package opentelemetry

type Config struct {
	Name      string
	Exporters []Exporter
}

type Exporter struct {
	Endpoint string
	Driver   string
	Enabled  bool
	Insecure bool
}
