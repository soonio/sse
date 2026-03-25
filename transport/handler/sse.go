package handler

import (
	"net/http"
	"strconv"
	"time"

	"go.uber.org/zap"

	"pusher/internal/core/hub"
	"pusher/internal/core/pool"
)

// SSE 处理 SSE 长连接监听。
func SSE(logger *zap.Logger, h *hub.Hub, mp *pool.MessagePool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()

		uid := strconv.FormatInt(r.Context().Value("UID").(int64), 10)

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		quit := make(chan struct{})
		timestamp := time.Now().UnixNano()
		h.Store(uid, timestamp, w, quit)
		defer h.Remove(uid, timestamp)

		timeout := time.NewTimer(30 * time.Minute)
		defer timeout.Stop()

		go func() {
			if c, ok := h.Get(uid); ok {
				msg := mp.Get()
				defer mp.Put(msg)
				msg.Body = uid
				msg.Timestamp = time.Now().Unix()
				if err := c.Push(msg); err != nil {
					logger.Error(err.Error())
				}
			}
		}()

		select {
		case <-quit:
			return
		case <-timeout.C:
			return
		case <-r.Context().Done():
			return
		}
	}
}
