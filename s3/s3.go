package s3

import (
	"context"
	"log/slog"
	"sync"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gowok/fp/maps"
	"github.com/gowok/fp/some"
	"github.com/gowok/gowok"
)

var clients = sync.Map{}

func Client(names ...string) some.Some[*s3.Client] {
	if len(names) == 0 {
		names = append(names, "default")
	}

	for _, name := range names {
		val, ok := clients.Load(name)
		if !ok {
			continue
		}
		client, ok := val.(*s3.Client)
		if !ok {
			continue
		}
		if client != nil {
			return some.Of(client)
		}
	}
	return some.Empty[*s3.Client]()
}

type ConfigureInput struct {
	OnError func(error)
}

func Configure(input *ConfigureInput) func(context.Context) {
	if input == nil {
		input = &ConfigureInput{}
	}

	if input.OnError == nil {
		input.OnError = func(err error) {
			slog.Error(err.Error())
		}
	}
	return func(ctx context.Context) {
		configMap := maps.Get(gowok.Config.Map(), "s3", map[string]any{})

		for key, val := range configMap {
			c := Config{}
			err := maps.ToStruct(val.(map[string]any), &c)
			if err != nil {
				continue
			}

			if !c.Enabled {
				continue
			}

			cfg, err := config.LoadDefaultConfig(ctx,
				config.WithRegion(c.Region),
			)
			if err != nil {
				input.OnError(err)
				return
			}

			if c.Endpoint != "" {
				cfg.BaseEndpoint = &c.Endpoint
			}

			s3Client := s3.NewFromConfig(cfg)

			_, err = s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
			if err != nil {
				input.OnError(err)
				return
			}

			clients.Store(key, s3Client)
		}
	}
}
