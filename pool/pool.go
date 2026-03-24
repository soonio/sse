package pool

import (
	"bytes"
	"sync"

	"pusher/types"
)

const maxBufferSize = 4 * 1024

type BufferPool struct {
	pool sync.Pool
}

func NewBufferPool(bufferSize int) *BufferPool {
	return &BufferPool{
		pool: sync.Pool{
			New: func() interface{} {
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

type MessagePool struct {
	pool sync.Pool
}

func NewMessagePool() *MessagePool {
	return &MessagePool{
		pool: sync.Pool{
			New: func() interface{} {
				return &types.Message[string]{Type: types.MessageTypePing, Timestamp: 0}
			},
		},
	}
}

func (p *MessagePool) Get() *types.Message[string] {
	return p.pool.Get().(*types.Message[string])
}

func (p *MessagePool) Put(msg *types.Message[string]) {
	msg.Timestamp = 0
	msg.Body = ""
	p.pool.Put(msg)
}
