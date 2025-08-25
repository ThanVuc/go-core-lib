package cache

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	Client *redis.Client
	Ctx    context.Context
	Cancel context.CancelFunc
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
		Client: rdb,
		Ctx:    ctx,
		Cancel: cancel,
	}
}

func Set[T any](r *RedisCache, key string, value T, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.Client.Set(r.Ctx, key, data, expiration).Err()
}

func Get[T any](r *RedisCache, key string) (T, error) {
	var result T

	data, err := r.Client.Get(r.Ctx, key).Result()
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
	count, err := r.Client.Exists(r.Ctx, key).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func Delete(r *RedisCache, key string) error {
	return r.Client.Del(r.Ctx, key).Err()
}

func (r *RedisCache) Close(wg *sync.WaitGroup) error {
	r.Cancel()
	defer wg.Done()
	return r.Client.Close()
}

func (r *RedisCache) Ping() error {
	return r.Client.Ping(r.Ctx).Err()
}
