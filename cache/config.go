package cache

type Config struct {
	Addr     string
	Password string
	DB       int
	PoolSize int
	MinIdle  int
}
