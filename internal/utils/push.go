package utils

import (
	"log"
	"net/http"
)

func Push(w http.ResponseWriter, msg string) {
	defer func() {
		if err := recover(); err != nil {
			log.Println(err)
		}
	}()
	if w == nil { // 可能被关闭了,就不推送了
		return
	}
	_, _ = w.Write([]byte("data: " + msg + "\n\n")) // 消息的固定格式
	w.(http.Flusher).Flush()
}
