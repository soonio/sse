package task

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"sse/internal/svc"
	"sse/internal/utils"

	"github.com/redis/go-redis/v9"
)

type Dispatcher struct {
	C           *svc.Container
	concurrency int
}

func NewDispatcher(c *svc.Container) *Dispatcher {
	return &Dispatcher{C: c, concurrency: 100}
}

func (d *Dispatcher) Run() {
	var ps = d.C.Red.Subscribe(context.Background(), d.C.Alive.Channel())
	defer func(ps *redis.PubSub) { _ = ps.Close() }(ps)

	cc := make(chan struct{}, d.concurrency)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-quit:
			return
		case msg := <-ps.Channel():
			cc <- struct{}{}
			go func(hub *svc.Hub, payload string) {
				defer func() { <-cc }()

				// 切掉retry
				Retried := strings.HasPrefix("retry", payload)
				if Retried {
					payload = strings.TrimPrefix("retry", payload)
				}

				m := strings.SplitN(payload, "|", 2)
				if len(m) == 2 {
					if w, ok := hub.Get(m[0]); ok {
						utils.Push(w, m[1])
					} else { // 未找到，可能切换了, 重试
						if !Retried { // 重试过的消息，不再重试
							go func(deliver *svc.Deliver, uid string, msg string) {
								_ = deliver.DispatchWithRetry(context.Background(), uid, msg, 5)
							}(d.C.Deliver, "retry"+m[0], m[1])
						}

					}
				}
			}(d.C.WriterHub, msg.Payload)
		}
	}
}

// Test 测试方法
func (d *Dispatcher) Test() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	var frq time.Duration = 1 // 每60单位ping一下

	var t = time.NewTicker(frq * time.Second)

	var i = 0
	for {
		select {
		case <-quit:
			fmt.Println("退出 test")
			return
		case <-t.C:
			t.Reset(frq * time.Second)
			i++
			err := d.C.Red.Publish(context.Background(), d.C.Alive.Channel(), fmt.Sprintf("Message %d", i)).Err()
			if err != nil {
				fmt.Println("Publish error:", err)
			}
		}
	}
}
