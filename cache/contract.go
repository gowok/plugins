package cache

import (
	"context"
)

type Client interface {
	PingContext(ctx context.Context) error
	Close() error
}

type ReaderContext[T any] interface {
	GetContext(ctx context.Context, key string) (T, error)
}

type Reader[T any] interface {
	Get(key string) (T, error)
}

type WriterContext[T any] interface {
	SetContext(ctx context.Context, key string, value T) error
}

type Writer[T any] interface {
	Set(key string, value T) error
}
