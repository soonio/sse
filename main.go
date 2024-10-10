package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var m = make(map[string]http.ResponseWriter)
var locker sync.Mutex

// 1 注册一个本机的全局MAP 用于记录用户 writer
// 2 注册一个redis MAP 用于记录用户在那台设备上

func sseHandler(w http.ResponseWriter, r *http.Request) {
	// 设置Content-Type为text/event-stream
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	uid := r.URL.Query().Get("uid")
	if uid == "" { // 没有用户ID 直接关闭
		_, _ = w.Write([]byte("data: 用户认证失败 " + time.Now().Format("2006-01-02 15:04:05") + "\n\n"))
		w.(http.Flusher).Flush()
		w.Header().Set("Connection", "close")

		fmt.Println("done2")
	} else {

		_, _ = w.Write([]byte("data: " + time.Now().Format("2006-01-02 15:04:05") + "\n\n"))
		w.(http.Flusher).Flush()

		locker.Lock()
		if wo, ok := m[uid]; ok {
			// 已经登陆一个了，直接关掉
			wo.Header().Set("Connection", "close")
		}
		m[uid] = w
		locker.Unlock()

		// 等待客户端关闭连接
		<-r.Context().Done()
		locker.Lock()
		delete(m, uid)
		locker.Unlock()

	}
}

func main() {
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		var t = time.NewTicker(60 * time.Second)

		for {
			select {
			case <-quit:
				fmt.Println("退出 ping")
				return
			case <-t.C:
				t.Reset(60 * time.Second)
				for s, w := range m {
					_, _ = w.Write([]byte("data: ping [" + s + "]" + time.Now().Format("2006-01-02 15:04:05") + "\n\n"))
					w.(http.Flusher).Flush()
				}
			}
		}
	}()

	go func() {
		http.HandleFunc("/sse", sseHandler)
		http.HandleFunc("/tips", func(w http.ResponseWriter, r *http.Request) {

			uid := r.URL.Query().Get("uid")
			msg := r.URL.Query().Get("msg")
			if uid == "" {
				w.WriteHeader(422)
				_, _ = w.Write([]byte("参数错误"))
			} else {
				if wo, ok := m[uid]; ok {
					_, _ = wo.Write([]byte("data: " + msg + time.Now().Format("2006-01-02 15:04:05") + "\n\n"))
					wo.(http.Flusher).Flush()

					w.WriteHeader(200)
					_, _ = w.Write([]byte("ok"))
				} else {
					w.WriteHeader(500)
					_, _ = w.Write([]byte("用户" + uid + "不存在"))
				}

			}

		})
		_ = http.ListenAndServe(":8080", nil)

	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("关闭主流程")
}
