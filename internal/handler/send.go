package handler

import (
	"net/http"

	"sse/internal/svc"
)

func Send(ctx *svc.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid := r.URL.Query().Get("uid")
		msg := r.URL.Query().Get("msg")

		if uid == "" {
			w.WriteHeader(422)
			_, _ = w.Write([]byte("参数错误"))
		} else {
			err := ctx.Deliver.DispatchWithRetry(r.Context(), uid, msg, 5)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(err.Error()))
			} else {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("ok"))
			}
		}
	}
}
