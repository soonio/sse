package svc

import (
	"net/http"
	"sync"
)

type Hub struct {
	m      map[string]http.ResponseWriter // 响应写入器
	locker sync.Mutex
}

func NewHub() *Hub {
	return &Hub{
		m:      make(map[string]http.ResponseWriter),
		locker: sync.Mutex{},
	}
}

func (w *Hub) Store(uid string, rw http.ResponseWriter) {
	w.locker.Lock()
	defer w.locker.Unlock()

	if wo, ok := w.m[uid]; ok {
		if wo != nil {
			wo.Header().Set("Connection", "close")
		}
	}
	w.m[uid] = rw
}

func (w *Hub) Remove(uid string) {
	w.locker.Lock()
	defer w.locker.Unlock()

	delete(w.m, uid)
}

func (w *Hub) Get(uid string) (http.ResponseWriter, bool) {
	//w.locker.Lock()
	//defer w.locker.Unlock()
	v, ok := w.m[uid]
	return v, ok
}
