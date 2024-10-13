package svc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// 通道管理，基于redis，主要用来进行唯一通道名称管理
// 在多机部署的时候，可以区分用户的链接在哪一个通道上

type Channel struct {
	prefix string //
	red    *redis.Client
	name   string
}

func MustNewChannel(prefix string, red *redis.Client) *Channel {
	no, err := red.Incr(context.Background(), fmt.Sprintf("%s:channel-no", prefix)).Result()
	if err != nil {
		panic(err)
	}

	return &Channel{
		prefix: prefix,
		red:    red,
		name:   fmt.Sprintf("%s:%d", prefix, no),
	}
}

func (c *Channel) Name() string {
	return c.name
}

func (c *Channel) Lock(ctx context.Context) error {
	ok, err := c.red.SetNX(ctx, c.uniqueKey(), time.Now().Format(time.DateTime), 0).Result()
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("channel not unique")
	}
	return nil
}

func (c *Channel) MustLock(ctx context.Context) {
	if err := c.Lock(ctx); err != nil {
		panic(err)
	}
}

func (c *Channel) Unlock(ctx context.Context) error {
	return c.red.Del(ctx, c.uniqueKey()).Err()
}

func (c *Channel) uniqueKey() string {
	return c.name + "_unique"
}
