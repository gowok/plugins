package cache

import (
	"errors"

	"github.com/gowok/fp/maps"
	"github.com/gowok/gowok"
)

type Config struct {
	Driver  string
	DSN     string
	Name    string
	Enabled bool
}

func (c Config) ConnectionName(fallback string) string {
	if c.Name != "" {
		return c.Name
	}
	return fallback
}

type Configs map[string]Config

func ConfigFromMap(configMap map[string]any) (Configs, error) {
	c := make(Configs)
	if err := maps.ToStruct(configMap, &c); err != nil {
		return nil, err
	}
	return c, nil
}

func ConfigFromProject() (Configs, error) {
	configAny, ok := gowok.Config.Map()["cache"]
	if !ok {
		return nil, errors.New("no configuration")
	}
	configMap, ok := configAny.(map[string]any)
	if !ok {
		return nil, errors.New("no configuration")
	}
	return ConfigFromMap(configMap)
}
