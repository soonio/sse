package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

var (
	addr        = flag.String("addr", "http://127.0.0.1:8913", "服务地址")
	connections = flag.Int("c", 1000, "并发连接数")
	duration    = flag.Duration("d", 0, "持续时间(0表示持续运行)")
	uidStart    = flag.Int64("uid", 1, "起始用户ID")
	expire      = flag.Duration("expire", 24*time.Hour, "token过期时间")
)

func main() {
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\n正在关闭连接...")
		cancel()
	}()

	if *duration > 0 {
		go func() {
			time.Sleep(*duration)
			cancel()
		}()
	}

	var (
		successCount atomic.Int64
		failCount    atomic.Int64
		msgCount     atomic.Int64
		wg           sync.WaitGroup
	)

	fmt.Printf("开始压测: %s, 连接数: %d, 起始UID: %d\n", *addr, *connections, *uidStart)
	fmt.Println("按 Ctrl+C 停止")
	fmt.Println("========================================")

	startTime := time.Now()

	for i := 0; i < *connections; i++ {
		select {
		case <-ctx.Done():
			break
		default:
		}

		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			uid := *uidStart + int64(id)
			token := generateJWT(uid, *expire)

			if connectSSE(ctx, *addr, token, &msgCount) {
				successCount.Add(1)
			} else {
				failCount.Add(1)
			}
		}(i)

		time.Sleep(time.Millisecond * 2)
	}

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		lastMsgCount := int64(0)
		lastTime := time.Now()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				now := time.Now()
				elapsed := now.Sub(lastTime).Seconds()
				currentMsg := msgCount.Load()
				msgRate := float64(currentMsg-lastMsgCount) / elapsed
				lastMsgCount = currentMsg
				lastTime = now

				fmt.Printf("[统计] 成功: %d, 失败: %d, 消息数: %d, 消息速率: %.0f/s, 运行时间: %s\n",
					successCount.Load(),
					failCount.Load(),
					currentMsg,
					msgRate,
					time.Since(startTime).Round(time.Second),
				)
			}
		}
	}()

	wg.Wait()

	fmt.Println("========================================")
	fmt.Printf("压测结束: 成功连接 %d, 失败 %d, 总消息 %d, 总耗时 %s\n",
		successCount.Load(),
		failCount.Load(),
		msgCount.Load(),
		time.Since(startTime).Round(time.Second),
	)
}

func generateJWT(uid int64, expire time.Duration) string {
	now := time.Now().Unix()
	header := map[string]string{
		"alg": "none",
		"typ": "JWT",
	}
	payload := map[string]interface{}{
		"uid": uid,
		"iat": now,
		"exp": now + int64(expire.Seconds()),
	}

	headerJSON, _ := json.Marshal(header)
	payloadJSON, _ := json.Marshal(payload)

	headerB64 := base64RawURLEncode(headerJSON)
	payloadB64 := base64RawURLEncode(payloadJSON)

	return fmt.Sprintf("%s.%s.", headerB64, payloadB64)
}

func base64RawURLEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func connectSSE(ctx context.Context, addr, token string, msgCount *atomic.Int64) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", addr+"/listen", nil)
	if err != nil {
		return false
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	client := &http.Client{
		Timeout: 0,
		Transport: &http.Transport{
			MaxIdleConns:        10000,
			MaxIdleConnsPerHost: 10000,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return false
	}

	go func() {
		defer resp.Body.Close()
		buf := make([]byte, 4096)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := resp.Body.Read(buf)
				if err != nil {
					if err != io.EOF {
					}
					return
				}
				if n > 0 {
					msgCount.Add(1)
				}
			}
		}
	}()

	return true
}
