package dispatch

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"pusher/hub"
	"pusher/pool"
	"pusher/types"
	"pusher/worker"
)

type Notifier interface {
	Push(string)
}

type Dispatcher struct {
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	logger    *zap.Logger
	red       *redis.Client
	stream    string
	group     string
	consumer  string
	notify    Notifier
	bp        pool.Pool[*bytes.Buffer]
	mp        pool.Pool[*types.Message[string]]
	hub       *hub.Hub
	pingPool  *sync.Pool
	pushPool  *worker.Pool
	pingPoolW *worker.Pool
}

func NewDispatcher(
	logger *zap.Logger,
	red *redis.Client,
	stream, group, consumer string,
	notify Notifier,
	bp pool.Pool[*bytes.Buffer],
	mp pool.Pool[*types.Message[string]],
	h *hub.Hub,
) *Dispatcher {
	ctx, cancel := context.WithCancel(context.Background())
	numWorkers := 100

	return &Dispatcher{
		ctx:      ctx,
		cancel:   cancel,
		logger:   logger,
		red:      red,
		stream:   stream,
		group:    group,
		consumer: consumer,
		notify:   notify,
		bp:       bp,
		mp:       mp,
		hub:      h,
		pingPool: &sync.Pool{
			New: func() any { return &types.Message[string]{Type: types.MessageTypePing, Timestamp: 0} },
		},
		pushPool:  worker.NewPool(numWorkers, 10000),
		pingPoolW: worker.NewPool(50, 10000),
	}
}

func (d *Dispatcher) Run() {
	d.wg.Add(1)
	go d.Ping()

	var args = &redis.XReadGroupArgs{
		Group:    d.group,
		Consumer: d.consumer,
		Streams:  []string{d.stream, ">"},
		Count:    3000,
		Block:    3 * time.Second,
	}

	for {
		select {
		case <-d.ctx.Done():
			return
		default:
			messages, err := d.red.XReadGroup(d.ctx, args).Result()
			if err != nil && !errors.Is(err, redis.Nil) {
				d.logger.Error(err.Error())
				select {
				case <-d.ctx.Done():
					return
				case <-time.After(5 * time.Second):
				}
				continue
			}

			if len(messages) == 0 {
				continue
			}

			for _, stream := range messages {
				d.wg.Add(1)
				ok := d.pushPool.Submit(func() {
					defer d.wg.Done()
					d.Push(stream.Messages)
				})

				if !ok {
					go func(msgs []redis.XMessage) {
						defer d.wg.Done()
						d.Push(msgs)
					}(stream.Messages)
				}

				var IDs = make([]string, 0, len(stream.Messages))
				for _, msg := range stream.Messages {
					IDs = append(IDs, msg.ID)
				}
				_, err = d.red.XAck(d.ctx, d.stream, d.group, IDs...).Result()
				if err != nil {
					d.logger.Error(err.Error())
				}
			}
		}
	}
}

func (d *Dispatcher) Ping() {
	defer d.wg.Done()

	var t = time.NewTicker(time.Minute)
	defer t.Stop()

	for {
		select {
		case <-d.ctx.Done():
			return
		case <-t.C:
			keys := d.hub.Keys()

			for _, uid := range keys {
				uid := uid
				ok := d.pingPoolW.Submit(func() {
					if w, ok := d.hub.Get(uid); ok {
						var msg = d.pingPool.Get().(*types.Message[string])
						_ = w.Push(msg)
						d.pingPool.Put(msg)
					}
				})

				if !ok {
					d.logger.Warn("心跳任务提交失败，队列已满", zap.String("uid", uid))
				}
			}
		}
	}
}

func (d *Dispatcher) Push(messages []redis.XMessage) {
	for _, message := range messages {
		uidVal, ok := message.Values["uid"]
		if !ok {
			d.logger.Warn("消息缺少 uid 字段", zap.String("id", message.ID))
			continue
		}
		uid, ok := uidVal.(string)
		if !ok {
			d.logger.Warn("uid 字段类型错误", zap.String("id", message.ID), zap.Any("uid", uidVal))
			continue
		}

		msgVal, ok := message.Values["msg"]
		if !ok {
			d.logger.Warn("消息缺少 msg 字段", zap.String("id", message.ID), zap.String("uid", uid))
			continue
		}
		msg, ok := msgVal.(string)
		if !ok {
			d.logger.Warn("msg 字段类型错误", zap.String("id", message.ID), zap.Any("msg", msgVal))
			continue
		}

		ok = d.pushPool.Submit(func() {
			if w, ok := d.hub.Get(uid); ok {
				tp := d.bp.Get()
				tp.WriteString(msg)
				err := w.Push(tp)
				if err != nil {
					if strings.HasSuffix(err.Error(), "panic:") {
						d.logger.Error(err.Error())
					}
					d.notify.Push(uid)
				}
				d.bp.Put(tp)
			} else {
				d.notify.Push(uid)
			}
		})

		if !ok {
			d.logger.Warn("推送任务提交失败，队列已满", zap.String("uid", uid))
		}
	}
}

func (d *Dispatcher) Close() {
	d.cancel()
	d.pushPool.Close()
	d.pingPoolW.Close()
	d.wg.Wait()
}
