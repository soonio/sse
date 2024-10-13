package svc

import (
	"sse/internal/config"

	"github.com/redis/go-redis/v9"
)

type Container struct {
	C   *config.Config
	Red *redis.Client

	WriterHub *Hub
	Alive     *Alive
	Deliver   *Deliver
}

func New(c *config.Config) *Container {
	var (
		prefix  = "sse:connect"
		red     = redis.NewClient(&redis.Options{Addr: c.Redis.Addr, Password: c.Redis.Password, DB: c.Redis.Database})
		channel = MustNewChannel(prefix, red)
		alive   = NewAlive(red, prefix, channel)
	)

	return &Container{
		C:         c,
		Red:       red,
		WriterHub: NewHub(),
		Alive:     alive,
		Deliver:   NewDeliver(red, alive.Channel()),
	}
}
