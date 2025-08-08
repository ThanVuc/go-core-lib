package cache

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
	ctx    context.Context
	cancel context.CancelFunc
}

func NewRedisCache(cfg Config) *RedisCache {
	ctx, cancel := context.WithCancel(context.Background())

	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdle,
	})

	return &RedisCache{
		client: rdb,
		ctx:    ctx,
		cancel: cancel,
	}
}

func Set[T any](r *RedisCache, key string, value T, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.client.Set(r.ctx, key, data, expiration).Err()
}

func Get[T any](r *RedisCache, key string) (T, error) {
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

func GetAndSet[T any](r *RedisCache, key string, value T, expiration time.Duration) (T, error) {
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

func Exists(r *RedisCache, key string) (bool, error) {
	count, err := r.client.Exists(r.ctx, key).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func Delete(r *RedisCache, key string) error {
	return r.client.Del(r.ctx, key).Err()
}

func (r *RedisCache) Close(wg *sync.WaitGroup) error {
	r.cancel()
	defer wg.Done()
	return r.client.Close()
}

func (r *RedisCache) Ping() error {
	return r.client.Ping(r.ctx).Err()
}
