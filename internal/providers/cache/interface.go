package cache

import (
	"context"
	"time"
)

type CacheProvider interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Del(ctx context.Context, keys ...string) error
	GetJSON(ctx context.Context, key string, out any) (bool, error)
	SetJSON(ctx context.Context, key string, val any, ttl time.Duration) error
}
