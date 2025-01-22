package gorm

type Config struct {
	Driver  string
	DSN     string
	Enabled bool
}

type Configs map[string]Config
