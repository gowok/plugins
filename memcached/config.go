package memcached

import "github.com/gowok/fp/maps"

type Config struct {
	Enabled bool
	Servers []string
}

type Configs map[string]Config

func ConfigFromMap(configMap map[string]any) Configs {
	c := make(Configs)
	_ = maps.ToStruct(configMap, &c)
	return c
}
