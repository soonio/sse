package types

import "encoding/json"

type Message struct {
	Type      string `json:"type"`      // 消息类型
	Body      string `json:"body"`      // 消息内容
	Timestamp int64  `json:"timestamp"` // 时间戳
}

func (m *Message) String() string {
	bs, _ := json.Marshal(m)
	return string(bs)
}
