//go:build wireinject
// +build wireinject

package app

import (
	"github.com/google/wire"
	"go.uber.org/zap"

	"pusher/config"
	"pusher/infrastructure/limiter"
	"pusher/infrastructure/pool"
	"pusher/infrastructure/redis"
	"pusher/internal/hub"
	"pusher/internal/service/notify"
	"pusher/internal/service/push"
	"pusher/monitor"
	"pusher/transport"
)

// ProviderSet 汇聚所有 wire provider。
var ProviderSet = wire.NewSet(
	// infrastructure
	redis.NewClient,

	// hub
	hub.NewHub,
	limiter.NewLimiter,
	provideBufferPool,
	pool.NewMessagePool,

	// service
	notify.NewService,
	wire.Bind(new(push.Notifier), new(*notify.Service)),
	push.NewService,

	// observability
	monitor.NewMonitor,
	wire.Bind(new(monitor.Counter), new(*hub.Hub)),

	// transport
	provideRouter,

	// app
	provideHTTPServer,
	wire.Struct(new(App), "*"),
)

func provideBufferPool() *pool.BufferPool {
	return pool.NewBufferPool(128)
}

func provideRouter(c *config.Config, logger *zap.Logger, h *hub.Hub, lim *limiter.Limiter, mp *pool.MessagePool) *transport.Router {
	return transport.NewRouter(c.Debug, logger, h, lim, mp)
}

func provideHTTPServer(c *config.Config, router *transport.Router) *Server {
	return NewHTTPServer(c.Addr, router)
}

// InitApp wire 入口，由 wire 工具自动生成 wire_gen.go。
func InitApp(c *config.Config, logger *zap.Logger) (*App, error) {
	wire.Build(ProviderSet)
	return nil, nil
}
