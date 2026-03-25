package push

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"pusher/config"
	"pusher/internal/core/hub"
	"pusher/internal/core/pool"
	"pusher/internal/core/worker"
	"pusher/types"
)

// Notifier 是离线通知接口，由 notify.Service 实现。
type Notifier interface {
	Push(string)
}

type Service struct {
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	logger    *zap.Logger
	red       *redis.Client
	stream    string
	group     string
	consumer  string
	notify    Notifier
	bp        *pool.BufferPool
	mp        *pool.MessagePool
	hub       *hub.Hub
	pingPool  *sync.Pool
	pushPool  *worker.Pool
	pingPoolW *worker.Pool
}

func NewService(
	logger *zap.Logger,
	red *redis.Client,
	notify Notifier,
	bp *pool.BufferPool,
	mp *pool.MessagePool,
	h *hub.Hub,
) *Service {
	ctx, cancel := context.WithCancel(context.Background())
	numWorkers := 100

	return &Service{
		ctx:      ctx,
		cancel:   cancel,
		logger:   logger,
		red:      red,
		stream:   config.StreamName,
		group:    config.ConsumerGroup,
		consumer: config.ConsumerNamePatten,
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

func (s *Service) Run() {
	s.wg.Add(1)
	go s.ping()

	var args = &redis.XReadGroupArgs{
		Group:    s.group,
		Consumer: s.consumer,
		Streams:  []string{s.stream, ">"},
		Count:    3000,
		Block:    3 * time.Second,
	}

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			messages, err := s.red.XReadGroup(s.ctx, args).Result()
			if err != nil && !errors.Is(err, redis.Nil) {
				s.logger.Error(err.Error())
				select {
				case <-s.ctx.Done():
					return
				case <-time.After(5 * time.Second):
				}
				continue
			}

			if len(messages) == 0 {
				continue
			}

			for _, stream := range messages {
				s.wg.Add(1)
				ok := s.pushPool.Submit(func() {
					defer s.wg.Done()
					s.push(stream.Messages)
				})

				if !ok {
					go func(msgs []redis.XMessage) {
						defer s.wg.Done()
						s.push(msgs)
					}(stream.Messages)
				}

				var IDs = make([]string, 0, len(stream.Messages))
				for _, msg := range stream.Messages {
					IDs = append(IDs, msg.ID)
				}
				_, err = s.red.XAck(s.ctx, s.stream, s.group, IDs...).Result()
				if err != nil {
					s.logger.Error(err.Error())
				}
			}
		}
	}
}

func (s *Service) ping() {
	defer s.wg.Done()

	var t = time.NewTicker(time.Minute)
	defer t.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-t.C:
			keys := s.hub.Keys()

			for _, uid := range keys {
				uid := uid
				ok := s.pingPoolW.Submit(func() {
					if w, ok := s.hub.Get(uid); ok {
						var msg = s.pingPool.Get().(*types.Message[string])
						_ = w.Push(msg)
						s.pingPool.Put(msg)
					}
				})

				if !ok {
					s.logger.Warn("心跳任务提交失败，队列已满", zap.String("uid", uid))
				}
			}
		}
	}
}

func (s *Service) push(messages []redis.XMessage) {
	for _, message := range messages {
		uidVal, ok := message.Values["uid"]
		if !ok {
			s.logger.Warn("消息缺少 uid 字段", zap.String("id", message.ID))
			continue
		}
		uid, ok := uidVal.(string)
		if !ok {
			s.logger.Warn("uid 字段类型错误", zap.String("id", message.ID), zap.Any("uid", uidVal))
			continue
		}

		msgVal, ok := message.Values["msg"]
		if !ok {
			s.logger.Warn("消息缺少 msg 字段", zap.String("id", message.ID), zap.String("uid", uid))
			continue
		}
		msg, ok := msgVal.(string)
		if !ok {
			s.logger.Warn("msg 字段类型错误", zap.String("id", message.ID), zap.Any("msg", msgVal))
			continue
		}

		ok = s.pushPool.Submit(func() {
			if w, ok := s.hub.Get(uid); ok {
				tp := s.bp.Get()
				tp.WriteString(msg)
				err := w.Push(tp)
				if err != nil {
					if strings.HasSuffix(err.Error(), "panic:") {
						s.logger.Error(err.Error())
					}
					s.notify.Push(uid)
				}
				s.bp.Put(tp)
			} else {
				s.notify.Push(uid)
			}
		})

		if !ok {
			s.logger.Warn("推送任务提交失败，队列已满", zap.String("uid", uid))
		}
	}
}

func (s *Service) RedisClient() *redis.Client {
	return s.red
}

func (s *Service) Close() {
	s.cancel()
	s.pushPool.Close()
	s.pingPoolW.Close()
	s.wg.Wait()
}
