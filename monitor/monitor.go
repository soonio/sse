package monitor

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"go.uber.org/zap"
)

type Counter interface {
	Len() int
}

type Monitor struct {
	ctx     context.Context
	cancel  context.CancelFunc
	logger  *zap.Logger
	counter Counter
}

func NewMonitor(logger *zap.Logger, counter Counter) *Monitor {
	ctx, cancel := context.WithCancel(context.Background())
	return &Monitor{
		ctx:     ctx,
		cancel:  cancel,
		logger:  logger,
		counter: counter,
	}
}

func (m *Monitor) Start() {
	interval := 5 * time.Second

	now := time.Now()
	next := now.Truncate(interval).Add(interval)
	delay := next.Sub(now)

	select {
	case <-m.ctx.Done():
		return
	case <-time.After(delay):
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var lastMemPrint time.Time

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.logger.Info("在线人数", zap.Int("num", m.counter.Len()))

			if time.Since(lastMemPrint) >= time.Minute {
				m.printMemStats()
				lastMemPrint = time.Now()
			}
		}
	}
}

func (m *Monitor) printMemStats() {
	var x runtime.MemStats
	runtime.ReadMemStats(&x)

	mb := func(d uint64) string {
		return fmt.Sprintf("%.2fMi", float64(d)/1024/1024)
	}

	m.logger.Info("内存统计",
		zap.String("Alloc", mb(x.Alloc)),
		zap.String("Sys", mb(x.Sys)),
		zap.String("HeapAlloc", mb(x.HeapAlloc)),
		zap.Uint32("NumGC", x.NumGC),
		zap.Int("Goroutines", runtime.NumGoroutine()),
	)
}

func (m *Monitor) Stop() {
	m.cancel()
}
