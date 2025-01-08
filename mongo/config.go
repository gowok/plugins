package mongo

import "github.com/gowok/gowok/maps"

type Config struct {
	DSN     string
	Enabled bool
	With    map[string]string
}

type Configs map[string]Config

func ConfigFromMap(configMap map[string]any) Configs {
	c := make(Configs)
	_ = maps.MapToStruct(configMap, &c)
	return c
}
