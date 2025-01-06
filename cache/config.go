package cache

import (
	"errors"

	"github.com/gowok/gowok"
	"github.com/gowok/gowok/maps"
)

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

func ConfigFromProject(project *gowok.Project) (Configs, error) {
	configAny, ok := project.ConfigMap["cache"]
	if !ok {
		return nil, errors.New("no configuration")
	}
	configMap, ok := configAny.(map[string]any)
	if !ok {
		return nil, errors.New("no configuration")
	}
	config := ConfigFromMap(configMap)
	return config, nil
}
