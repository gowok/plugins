package redis

import (
	"bytes"
	"context"
	"encoding/gob"

	"github.com/redis/go-redis/v9"
)

type client struct {
	redisClient *redis.Client
}

func (c *client) Set(ctx context.Context, key string, value any) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(&value); err != nil {
		return err
	}
	return c.redisClient.Set(ctx, key, buf.Bytes(), 0).Err()
}

func (c *client) Get(ctx context.Context, key string) (any, error) {
	data, err := c.redisClient.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}

	var value any
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&value); err != nil {
		return nil, err
	}
	return value, nil
}

func (c *client) Delete(ctx context.Context, key string) error {
	return c.redisClient.Del(ctx, key).Err()
}
