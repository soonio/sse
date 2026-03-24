package limiter

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

type limiterItem struct {
	valid time.Time
}

type Limiter struct {
	locker   sync.RWMutex
	store    map[string]*limiterItem
	ctx      context.Context
	cancel   context.CancelFunc
	count    atomic.Int64
	maxCount int64
}

func NewLimiter() *Limiter {
	return &Limiter{
		store:    make(map[string]*limiterItem),
		maxCount: 50000,
	}
}

func (m *Limiter) Have(key string, seconds int) error {
	m.locker.Lock()
	defer m.locker.Unlock()

	if v, ok := m.store[key]; ok {
		if v.valid.After(time.Now()) {
			return errors.New("repeated key")
		}
	}
	m.store[key] = &limiterItem{
		valid: time.Now().Add(time.Duration(seconds) * time.Second),
	}
	m.count.Add(1)
	return nil
}

func (m *Limiter) Has(key string) bool {
	m.locker.Lock()
	defer m.locker.Unlock()
	if item, ok := m.store[key]; ok {
		if item.valid.After(time.Now()) {
			return true
		}
		delete(m.store, key)
		m.count.Add(-1)
	}
	return false
}

func (m *Limiter) rebuild() {
	m.locker.Lock()
	defer m.locker.Unlock()

	now := time.Now()
	newStore := make(map[string]*limiterItem)
	var validCount int64

	for k, v := range m.store {
		if v.valid.After(now) {
			newStore[k] = v
			validCount++
		}
	}

	m.store = newStore
	m.count.Store(validCount)
}

func (m *Limiter) cleanup() {
	m.locker.Lock()
	defer m.locker.Unlock()

	now := time.Now()
	for k, v := range m.store {
		if v.valid.Before(now) {
			delete(m.store, k)
			m.count.Add(-1)
		}
	}
}

func (m *Limiter) Start() {
	m.ctx, m.cancel = context.WithCancel(context.Background())

	cleanupTicker := time.NewTicker(10 * time.Second)
	defer cleanupTicker.Stop()

	rebuildTicker := time.NewTicker(5 * time.Minute)
	defer rebuildTicker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-cleanupTicker.C:
			m.cleanup()
		case <-rebuildTicker.C:
			if m.count.Load() > m.maxCount {
				m.rebuild()
			}
		}
	}
}

func (m *Limiter) Close() {
	m.cancel()
}

func (m *Limiter) Len() int64 {
	return m.count.Load()
}
