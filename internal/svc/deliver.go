package svc

import (
	"context"
	"errors"
	"github.com/redis/go-redis/v9"
	"time"
)

type Deliver struct {
	red       *redis.Client
	script    *redis.Script
	durations []time.Duration
	channel   string
}

func NewDeliver(red *redis.Client, channel string) *Deliver {
	return &Deliver{
		red:    red,
		script: redis.NewScript(`local ccc = redis.call('GET', KEYS[1]); if ccc then redis.call('PUBLISH', ccc, ARGV[1]);return 1; else return 0 end`),
		durations: []time.Duration{
			500 * time.Millisecond,
			1000 * time.Millisecond,
			1000 * time.Millisecond,
			2000 * time.Millisecond,
			3000 * time.Millisecond,
		},
		channel: channel,
	}
}

// Dispatch 不带重试的推送
func (d *Deliver) Dispatch(ctx context.Context, uid, msg string) (bool, error) {
	code, err := d.script.Run(ctx, d.red, []string{d.channel + ":" + uid}, uid+"|"+msg).Result()
	if err != nil {
		return false, err
	}
	return code.(int64) == 1, nil
}

// DispatchWithRetry 带有重试的推送
func (d *Deliver) DispatchWithRetry(ctx context.Context, uid, msg string, attempts int) error {
	attempts = min(attempts, len(d.durations))
	attempts = max(attempts, 0)

	for i := -1; i < attempts; i++ {
		if i != -1 { // 正常的第一跳过
			time.Sleep(d.durations[i])
		}
		ok, err := d.Dispatch(ctx, uid, msg)
		if err != nil { // 有错误，很有可能redis挂了或者脚本有错误，不进行重试
			return err
		}
		if ok {
			return nil
		}
	}
	return errors.New("retry limit exceeded")
}
