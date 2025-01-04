package cache

import "github.com/gowok/gowok/maps"

type Config struct {
	Driver  string
	DSN     string
	Enabled bool
}

type Configs map[string]Config

func ConfigFromMap(configMap map[string]any) Configs {
	c := make(Configs)
	maps.MapToStruct(configMap, &c)
	return c
}
