package redis

import (
	"time"

	"github.com/redis/go-redis/v9"

	"pusher/config"
)

func NewClient(c *config.Config) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:            c.Redis.Addr,
		Password:        c.Redis.Password,
		DB:              c.Redis.Database,
		PoolSize:        120,
		MinIdleConns:    64,
		MaxIdleConns:    128,
		ConnMaxIdleTime: 5 * time.Minute,
		ConnMaxLifetime: 10 * time.Minute,
	})
}
