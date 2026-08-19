// Package store implements data access for PostgreSQL and Redis.
package store

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisAdapter wraps go-redis to implement the service.RedisClient interface.
type RedisAdapter struct {
	client *redis.Client
}

// NewRedisAdapter creates a new Redis adapter.
func NewRedisAdapter(client *redis.Client) *RedisAdapter {
	return &RedisAdapter{client: client}
}

// Get retrieves a key from Redis.
func (r *RedisAdapter) Get(ctx context.Context, key string) (string, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("key not found")
	}
	return val, err
}

// Set stores a key-value pair with TTL.
func (r *RedisAdapter) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

// Del removes one or more keys.
func (r *RedisAdapter) Del(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}

// Exists checks if keys exist. Returns the count of existing keys.
func (r *RedisAdapter) Exists(ctx context.Context, keys ...string) (int64, error) {
	return r.client.Exists(ctx, keys...).Result()
}

// Incr atomically increments a key and returns the new value.
func (r *RedisAdapter) Incr(ctx context.Context, key string) (int64, error) {
	return r.client.Incr(ctx, key).Result()
}

// Expire sets a TTL on a key.
func (r *RedisAdapter) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return r.client.Expire(ctx, key, ttl).Err()
}
