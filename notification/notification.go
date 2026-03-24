package notification

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/soonio/bundle"
	"go.uber.org/zap"
)

type Notification struct {
	logger *zap.Logger
	batch  *bundle.Bundle[string]
	client *http.Client
}

func NewNotification(logger *zap.Logger) *Notification {
	var n = &Notification{logger: logger}
	n.batch = bundle.New(
		n.Send,
		bundle.WithSize[string](1000),
		bundle.WithTimeout[string](3*time.Second),
		bundle.WithPayloadSize[string](100000),
	)
	n.client = &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	return n
}

func (n *Notification) Send(UIDs []string) {
	n.logger.Info(
		"推送消息",
		zap.Strings("UIDs", UIDs),
	)
}

func (n *Notification) Push(uid string) {
	n.batch.Add(uid)
}

func (n *Notification) Start() {
	n.batch.Start()
}

func (n *Notification) Close() {
	n.batch.Close()
}
