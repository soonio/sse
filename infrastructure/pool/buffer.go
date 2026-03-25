package pool

import (
	"bytes"
	"sync"
)

const maxBufferSize = 4 * 1024

type BufferPool struct {
	pool sync.Pool
}

func NewBufferPool(bufferSize int) *BufferPool {
	return &BufferPool{
		pool: sync.Pool{
			New: func() any {
				buf := new(bytes.Buffer)
				return buf
			},
		},
	}
}

func (p *BufferPool) Get() *bytes.Buffer {
	buf := p.pool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

func (p *BufferPool) Put(buf *bytes.Buffer) {
	if buf.Cap() > maxBufferSize {
		return
	}
	buf.Reset()
	p.pool.Put(buf)
}
