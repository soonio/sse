package handler

import (
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"sse/internal/svc"
	"sse/internal/types"
	"sse/internal/utils"
)

func Connect(ctx *svc.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO 这里应该做一个中间件，解析 TOKEN 返回用户ID
		var uid = (func(_ string) string {
			return strconv.FormatInt(rand.New(rand.NewSource(time.Now().Unix())).Int63n(99999)+100000, 10)
		})(r.Header.Get("Authorization"))

		// 注册自己所在队列
		if err := ctx.Alive.Set(uid); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// 删除自己所在队列
		defer func() { _ = ctx.Alive.Remove(uid) }()

		w.Header().Set("Content-Type", "text/event-stream") // 设置Content-Type为text/event-stream
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		// 注册到 hub 中
		ctx.WriterHub.Store(uid, w)
		// 从 hub 删除
		defer ctx.WriterHub.Remove(uid)

		utils.Push(w, (&types.Message{Type: "ping", Body: fmt.Sprintf("您的用户ID为 %s", uid), Timestamp: time.Now().Unix()}).String())
		<-r.Context().Done() // 等待客户端关闭连接
	}
}
