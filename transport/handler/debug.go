package handler

import (
	"bytes"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	jsoniter "github.com/json-iterator/go"

	"pusher/internal/core/hub"
)

// Online 返回当前在线用户列表（仅 debug 模式）。
func Online(h *hub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bs, _ := jsoniter.Marshal(h.Keys())

		w.Header().Set("content-type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(bs)
	}
}

// Send 向指定用户推送消息（仅 debug 模式）。
func Send(h *hub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid := chi.URLParam(r, "uid")

		msg, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		if uid == "" {
			w.WriteHeader(422)
			_, _ = w.Write([]byte("参数错误"))
		} else {
			if c, ok := h.Get(uid); ok {
				_ = c.Push(bytes.NewBuffer(msg))
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("ok"))
			} else {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("user offline"))
			}
		}
	}
}

// Close 占位关闭接口。
func Close() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}
}
