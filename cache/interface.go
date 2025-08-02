package cache

type RedisCache interface {
	Close() error
}
