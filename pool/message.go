package pool

import (
	"sync"

	"pusher/types"
)

type MessagePool struct {
	pool sync.Pool
}

func NewMessagePool() *MessagePool {
	return &MessagePool{
		pool: sync.Pool{
			New: func() any {
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
