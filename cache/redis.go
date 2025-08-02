package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisCache struct {
	client *redis.Client
	ctx    context.Context
	cancel context.CancelFunc
}

func NewRedisCache(cfg Config) RedisCache {
	ctx, cancel := context.WithCancel(context.Background())

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	return &redisCache{
		client: rdb,
		ctx:    ctx,
		cancel: cancel,
	}
}

func Set[T any](r *redisCache, key string, value T, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.client.Set(r.ctx, key, data, expiration).Err()
}

func Get[T any](r *redisCache, key string) (T, error) {
	var result T

	data, err := r.client.Get(r.ctx, key).Result()
	if err != nil {
		return result, err
	}

	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return result, err
	}

	return result, nil
}

func GetAndSet[T any](r *redisCache, key string, value T, expiration time.Duration) (T, error) {
	var result T

	existing, err := Get[T](r, key)
	if err == nil {
		return existing, nil
	}

	if err != redis.Nil {
		return result, err
	}

	if err := Set(r, key, value, expiration); err != nil {
		return result, err
	}

	return value, nil
}

func Exists(r *redisCache, key string) (bool, error) {
	count, err := r.client.Exists(r.ctx, key).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func Delete(r *redisCache, key string) error {
	return r.client.Del(r.ctx, key).Err()
}

func (r *redisCache) Close() error {
	r.cancel()
	return r.client.Close()
}
