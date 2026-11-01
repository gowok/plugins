package redis

import "github.com/gowok/fp/maps"

type Config struct {
	DSN     string
	Enabled bool
}

type Configs map[string]Config

func ConfigFromMap(configMap map[string]any) Configs {
	c := make(Configs)
	_ = maps.ToStruct(configMap, &c)
	return c
}
