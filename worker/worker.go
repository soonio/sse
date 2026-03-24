package worker

import (
	"context"
	"sync"
)

type Job func()

type Pool struct {
	ctx       context.Context
	cancel    context.CancelFunc
	jobQueue  chan Job
	workers   int
	waitGroup sync.WaitGroup
}

func NewPool(workers int, queueSize int) *Pool {
	ctx, cancel := context.WithCancel(context.Background())
	pool := &Pool{
		ctx:      ctx,
		cancel:   cancel,
		jobQueue: make(chan Job, queueSize),
		workers:  workers,
	}

	for i := 0; i < workers; i++ {
		pool.waitGroup.Add(1)
		go pool.worker()
	}

	return pool
}

func (p *Pool) worker() {
	defer p.waitGroup.Done()

	for {
		select {
		case job := <-p.jobQueue:
			job()
		case <-p.ctx.Done():
			return
		}
	}
}

func (p *Pool) Submit(job Job) bool {
	select {
	case p.jobQueue <- job:
		return true
	case <-p.ctx.Done():
		return false
	}
}

func (p *Pool) Close() {
	p.cancel()
	p.waitGroup.Wait()
	close(p.jobQueue)
}

func (p *Pool) Workers() int {
	return p.workers
}

func (p *Pool) QueueLen() int {
	return len(p.jobQueue)
}
