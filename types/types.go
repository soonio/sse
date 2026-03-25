package types

import (
	"bytes"
	"io"

	jsoniter "github.com/json-iterator/go"
)

const (
	MessageTypePing = "ping"
)

type Message[T any] struct {
	Type      string `json:"type"`
	Body      T      `json:"body,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

func (m *Message[T]) String() string {
	bs, _ := jsoniter.Marshal(m)
	return string(bs)
}

type Slug struct {
	ID int64 `json:"id"`
}

type Body map[string]any

func NewBody() Body {
	return make(Body)
}

func (b Body) Get(key string) any {
	return b[key]
}

func (b Body) Set(key string, value any) Body {
	b[key] = value
	return b
}

func (b Body) Del(key string) Body {
	delete(b, key)
	return b
}

func (b Body) Reader() io.Reader {
	bs, err := jsoniter.Marshal(b)
	if err != nil {
		panic(err)
	}
	return bytes.NewReader(bs)
}
