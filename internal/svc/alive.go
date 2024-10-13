package svc

import (
	"context"
	"github.com/redis/go-redis/v9"
)

type Alive struct {
	red     *redis.Client
	prefix  string
	channel *Channel
	set     *redis.Script
	remove  *redis.Script
}

func NewAlive(c *redis.Client, prefix string, channel *Channel) *Alive {
	return &Alive{
		red:     c,
		prefix:  prefix + ":alive:",
		channel: channel,
		set:     redis.NewScript(`redis.call('HSET', KEYS[3], KEYS[1], KEYS[2]); return redis.call('SET', KEYS[1], KEYS[2]);`),
		remove:  redis.NewScript(`local ccc = redis.call('GET', KEYS[1]); if ccc == ARGV[1] then redis.call('DEL', KEYS[1]);return 1; else return 0 end`),
	}
}

func (a *Alive) Channel() string {
	return a.prefix
}

func (a *Alive) MustLock(ctx context.Context) {
	a.channel.MustLock(ctx)
}

func (a *Alive) Unlock(ctx context.Context) error {
	return a.channel.Unlock(ctx)
}

func (a *Alive) Set(id string) error {
	return a.set.Run(context.Background(), a.red, []string{a.prefix + id, a.channel.Name(), a.channel.Name() + "-hub"}).Err()
}

// Remove 删除 key，使用 lua 脚本，只有自己创建的自己才能删除
func (a *Alive) Remove(id string) error {
	return a.remove.Run(context.Background(), a.red, []string{a.prefix + id}, a.channel.Name()).Err()
}

func (a *Alive) Destroy() {

}
