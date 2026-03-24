package hub

import (
	"errors"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type Connection struct {
	Writer    http.ResponseWriter
	Close     chan struct{}
	timestamp int64
	lock      sync.Mutex
}

func (c *Connection) Push(msg fmt.Stringer) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	c.lock.Lock()
	defer c.lock.Unlock()

	_, err = c.Writer.Write([]byte("data: " + msg.String() + "\n\n"))
	if err != nil {
		return
	}

	if f, ok := c.Writer.(http.Flusher); ok {
		f.Flush()
		return nil
	}
	return errors.New("flusher fail")
}

type Hub struct {
	m         map[string]*Connection
	lock      sync.RWMutex
	count     atomic.Int64
	rebuildCh chan struct{}
	closeCh   chan struct{}
}

func NewHub() *Hub {
	return &Hub{
		m:         make(map[string]*Connection),
		rebuildCh: make(chan struct{}, 1),
		closeCh:   make(chan struct{}),
	}
}

func (w *Hub) Store(uid string, timestamp int64, rw http.ResponseWriter, quit chan struct{}) {
	w.lock.Lock()

	if hi, exist := w.m[uid]; exist {
		close(hi.Close)
		w.count.Add(-1)
	}

	w.m[uid] = &Connection{Writer: rw, Close: quit, timestamp: timestamp}
	w.count.Add(1)

	w.lock.Unlock()
}

func (w *Hub) Remove(uid string, timestamp int64) {
	w.lock.Lock()
	if hi, ok := w.m[uid]; ok {
		if hi.timestamp == timestamp {
			delete(w.m, uid)
			w.count.Add(-1)
		}
	}
	w.lock.Unlock()
}

func (w *Hub) Get(uid string) (*Connection, bool) {
	w.lock.RLock()
	v, ok := w.m[uid]
	w.lock.RUnlock()
	return v, ok
}

func (w *Hub) Keys() []string {
	w.lock.RLock()
	defer w.lock.RUnlock()

	keys := make([]string, 0, len(w.m))
	for uid := range w.m {
		keys = append(keys, uid)
	}
	return keys
}

func (w *Hub) Len() int {
	return int(w.count.Load())
}

func (w *Hub) Clear() {
	close(w.closeCh)
	w.lock.Lock()
	defer w.lock.Unlock()
	for _, x := range w.m {
		select {
		case <-x.Close:
		default:
			close(x.Close)
		}
	}
	w.m = make(map[string]*Connection)
	w.count.Store(0)
}

func (w *Hub) Rebuild() {
	select {
	case w.rebuildCh <- struct{}{}:
	default:
	}
}

func (w *Hub) Start() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-w.closeCh:
			return
		case <-ticker.C:
			w.rebuild()
		case <-w.rebuildCh:
			w.rebuild()
		}
	}
}

func (w *Hub) rebuild() {
	w.lock.Lock()
	defer w.lock.Unlock()

	if len(w.m) == 0 {
		return
	}

	newMap := make(map[string]*Connection, len(w.m))
	for k, v := range w.m {
		newMap[k] = v
	}
	w.m = newMap
}
