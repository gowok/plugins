package grpc

import (
	"errors"

	"github.com/gowok/fp/maps"
	"github.com/gowok/gowok"
)

type Config struct {
	Enabled bool
	Host    string
}

func getConfig() (*Config, error) {
	configAny, ok := gowok.Config.Map()["grpc"]
	if !ok {
		return nil, errors.New("no configuration")
	}

	configMap, ok := configAny.(map[string]any)
	if !ok {
		return nil, errors.New("no configuration")
	}

	config := &Config{}
	err := maps.ToStruct(configMap, &config)
	if err != nil {
		return nil, err
	}

	return config, nil
}
