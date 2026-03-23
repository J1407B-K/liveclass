package ratelimit

import (
	"context"
	"errors"
	"time"

	"github.com/go-redis/redis/v8"
)

var redisBucketScript = redis.NewScript(`
local key = KEYS[1]
local rate = tonumber(ARGV[1])
local capacity = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local requested = tonumber(ARGV[4])
local ttl = tonumber(ARGV[5])

local bucket = redis.call("HMGET", key, "tokens", "ts")
local tokens = tonumber(bucket[1])
local last = tonumber(bucket[2])

if tokens == nil then
  tokens = capacity
  last = now
end

if last == nil then
  last = now
end

local delta = now - last
if delta < 0 then
  delta = 0
end

tokens = tokens + delta * rate
if tokens > capacity then
  tokens = capacity
end
last = now

local allowed = 0
if tokens >= requested then
  tokens = tokens - requested
  allowed = 1
end

redis.call("HMSET", key, "tokens", tokens, "ts", last)
redis.call("PEXPIRE", key, ttl)
return {allowed, tokens}
`)

// RedisTokenBucket is a distributed token bucket backed by Redis.
type RedisTokenBucket struct {
	client   *redis.Client
	key      string
	rate     float64
	capacity float64
	ttl      time.Duration
}

// NewRedisTokenBucket creates a Redis-backed limiter.
// rate means tokens per second, capacity is the bucket size.
func NewRedisTokenBucket(client *redis.Client, key string, rate float64, capacity int64, ttl time.Duration) (*RedisTokenBucket, error) {
	if client == nil {
		return nil, errors.New("nil redis client")
	}
	if key == "" {
		return nil, errors.New("empty redis key")
	}
	if rate <= 0 {
		return nil, errors.New("rate must be positive")
	}
	if capacity <= 0 {
		return nil, errors.New("capacity must be positive")
	}
	if ttl <= 0 {
		ttl = time.Minute
	}
	return &RedisTokenBucket{
		client:   client,
		key:      key,
		rate:     rate,
		capacity: float64(capacity),
		ttl:      ttl,
	}, nil
}

// Allow consumes n tokens if available.
func (tb *RedisTokenBucket) Allow(ctx context.Context, n int64) (bool, error) {
	if n <= 0 {
		return true, nil
	}
	if tb == nil {
		return false, errors.New("nil token bucket")
	}
	now := float64(time.Now().UnixNano()) / 1e9
	res, err := redisBucketScript.Run(
		ctx,
		tb.client,
		[]string{tb.key},
		tb.rate,
		tb.capacity,
		now,
		float64(n),
		tb.ttl.Milliseconds(),
	).Result()
	if err != nil {
		return false, err
	}
	values, ok := res.([]interface{})
	if !ok || len(values) == 0 {
		return false, errors.New("invalid redis script result")
	}
	allowed, ok := values[0].(int64)
	if !ok {
		return false, errors.New("invalid allow flag type")
	}
	return allowed == 1, nil
}

// AllowRedis is a helper to consume tokens with Redis-backed bucket without storing the instance.
func AllowRedis(ctx context.Context, client *redis.Client, key string, rate float64, capacity int64, tokens int64, ttl time.Duration) (bool, error) {
	tb, err := NewRedisTokenBucket(client, key, rate, capacity, ttl)
	if err != nil {
		return false, err
	}
	return tb.Allow(ctx, tokens)
}
