package s3

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gowok/fp/maps"
	"github.com/gowok/fp/some"
	"github.com/gowok/gowok"
	"github.com/gowok/gowok/storage"
	"github.com/ngamux/ngamux"
)

var clients = sync.Map{}

func Client(names ...string) some.Some[*client] {
	if len(names) == 0 {
		names = append(names, "default")
	}

	for _, name := range names {
		val, ok := clients.Load(name)
		if !ok {
			continue
		}
		client, ok := val.(*client)
		if !ok {
			continue
		}
		if client != nil {
			return some.Of(client)
		}
	}
	return some.Empty[*client]()
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

			s3Client := &client{
				Client: *s3.NewFromConfig(cfg),
				bucket: c.Bucket,
			}

			_, err = s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
			if err != nil {
				input.OnError(err)
				return
			}

			clients.Store(key, s3Client)
			storage.Register(key, s3Client)
		}

		gowok.Health.Add("s3", func() any {
			status := "UP"
			results := map[string]any{}

			clients.Range(func(key, value any) bool {
				name, ok := key.(string)
				if !ok {
					status = "DOWN"
					return true
				}

				c, ok := value.(*client)
				if !ok || c == nil {
					status = "DOWN"
					results[name] = ngamux.Map{"status": "DOWN"}
					return true
				}

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				_, err := c.HeadBucket(ctx, &s3.HeadBucketInput{
					Bucket: gowok.ToPtr(c.bucket),
				})
				cancel()
				if err != nil {
					status = "DOWN"
					results[name] = ngamux.Map{"status": "DOWN"}
				} else {
					results[name] = ngamux.Map{"status": "UP"}
				}
				return true
			})

			if len(results) == 0 {
				status = "DOWN"
			}
			return ngamux.Map{"status": status, "clients": results}
		})

	}
}
