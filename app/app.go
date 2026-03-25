package app

import (
	"net/http"

	"go.uber.org/zap"

	"pusher/internal/core/hub"
	"pusher/internal/core/limiter"
	"pusher/internal/service/notify"
	"pusher/internal/service/push"
	"pusher/monitor"
	"pusher/transport"
)

// Server 是 HTTP 服务器的包装类型。
type Server struct {
	*http.Server
}

// App 聚合所有顶层组件，用于启动和关闭。
type App struct {
	Logger    *zap.Logger
	Hub       *hub.Hub
	PushSvc   *push.Service
	NotifySvc *notify.Service
	Monitor   *monitor.Monitor
	Limiter   *limiter.Limiter
	HTTPSrv   *Server
}

// NewHTTPServer 构造 HTTP Server。
func NewHTTPServer(addr string, router *transport.Router) *Server {
	return &Server{&http.Server{Addr: addr, Handler: router.Mux}}
}
