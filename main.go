package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sse/internal/config"
	"sse/internal/handler"
	"sse/internal/svc"
	"sse/internal/task"
)

var configFile = flag.String("f", "config.yml", "the config file")

func main() {
	flag.Parse()

	time.Local, _ = time.LoadLocation("Asia/Shanghai") // 设置全局时区

	var c config.Config
	c.MustLoad(*configFile)

	var ctx = svc.New(&c)

	var cbg = context.Background()

	ctx.Alive.MustLock(cbg)                                   // 进行唯一性锁定
	defer func(a *svc.Alive) { _ = a.Unlock(cbg) }(ctx.Alive) // 解锁唯一性

	go func() { // 使用协程创建服务监听
		http.HandleFunc("/listen", handler.Connect(ctx))
		http.HandleFunc("/push", handler.Send(ctx))
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "404 迷路了吗", http.StatusNotFound)
		})
		_ = http.ListenAndServe(c.Addr, nil)
	}()

	go task.NewDispatcher(ctx).Run()

	fmt.Printf("ListenOn: %s...\n", c.Addr)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Printf("Close At: %s", time.Now().Format(time.DateTime))
}
