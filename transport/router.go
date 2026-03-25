package transport

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"pusher/infrastructure/limiter"
	"pusher/infrastructure/pool"
	"pusher/internal/hub"
	thandler "pusher/transport/handler"
	tmiddleware "pusher/transport/middleware"
)

// Router 封装 chi.Mux，供 wire 区分类型。
type Router struct {
	Mux *chi.Mux
}

// NewRouter 构建并返回 HTTP 路由。
func NewRouter(
	debug bool,
	logger *zap.Logger,
	h *hub.Hub,
	lim *limiter.Limiter,
	mp *pool.MessagePool,
) *Router {
	r := chi.NewRouter()

	r.Group(func(r chi.Router) {
		r.Use(tmiddleware.Auth(logger, lim))
		r.Get("/listen", thandler.SSE(logger, h, mp))
	})

	r.Mount("/eqw2025ddffxce", middleware.Profiler())

	if debug {
		r.Post("/online", thandler.Online(h))
		r.Post("/send/{uid}", thandler.Send(h))
		r.Post("/close", thandler.Close())
	}

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "404 迷路了吗", http.StatusNotFound)
	})

	return &Router{Mux: r}
}
