package mongo

import "github.com/gowok/fp/maps"

type Config struct {
	DSN     string
	Enabled bool
	With    map[string]string
}

type Configs map[string]Config

func ConfigFromMap(configMap map[string]any) Configs {
	c := make(Configs)
	_ = maps.ToStruct(configMap, &c)
	return c
}
