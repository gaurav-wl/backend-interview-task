package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// redisProvider implements the CacheProvider interface using the go-redis library.
type redisProvider struct {
	client *redis.Client
	logger *zap.Logger
}

// NewRedisCacheProvider creates and returns a redisProvider strucy that satisfies the CacheProvider interface.
func NewRedisCacheProvider(ctx context.Context, address string, password string, logger *zap.Logger) (CacheProvider, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     address,
		Password: password,
	})

	if _, err := rdb.Ping(ctx).Result(); err != nil {
		return nil, err
	}

	return &redisProvider{
		client: rdb,
		logger: logger,
	}, nil
}

// Get retrieves a value from Redis.
func (r *redisProvider) Get(ctx context.Context, key string) (string, error) {
	val, err := r.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil // Return empty string if key does not exist
	}
	return val, err
}

// Set stores a value in Redis with an expiration.
func (r *redisProvider) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}

// Del deletes one or more keys from Redis.
func (r *redisProvider) Del(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}

// GetJSON retrieves a JSON value from Redis and unmarshals it into the provided output.
func (r *redisProvider) GetJSON(ctx context.Context, key string, out any) (bool, error) {
	raw, err := r.Get(ctx, key)
	if err != nil {
		return false, err
	}
	if raw == "" {
		return false, nil
	}
	if err := json.Unmarshal([]byte(raw), out); err != nil {
		return false, err
	}
	return true, nil
}

// SetJSON marshals a value to JSON and stores it in Redis with an expiration.
func (r *redisProvider) SetJSON(ctx context.Context, key string, val any, ttl time.Duration) error {
	b, err := json.Marshal(val)
	if err != nil {
		return err
	}
	return r.Set(ctx, key, string(b), ttl)
}
