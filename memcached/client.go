package memcached

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"

	"github.com/bradfitz/gomemcache/memcache"
)

type client struct {
	memcacheClient *memcache.Client
}

func (c *client) Set(ctx context.Context, key string, value any) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(&value); err != nil {
		return err
	}
	return c.memcacheClient.Set(&memcache.Item{
		Key:   key,
		Value: buf.Bytes(),
	})
}

func (c *client) Get(ctx context.Context, key string) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	item, err := c.memcacheClient.Get(key)
	if err != nil {
		return nil, err
	}

	var value any
	if err := gob.NewDecoder(bytes.NewReader(item.Value)).Decode(&value); err != nil {
		return nil, err
	}
	return value, nil
}

func (c *client) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	err := c.memcacheClient.Delete(key)
	if errors.Is(err, memcache.ErrCacheMiss) {
		return nil
	}
	return err
}
