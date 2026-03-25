package notify

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/soonio/bundle"
	"go.uber.org/zap"
)

type Service struct {
	logger *zap.Logger
	batch  *bundle.Bundle[string]
	client *http.Client
}

func NewService(logger *zap.Logger) *Service {
	var s = &Service{logger: logger}
	s.batch = bundle.New(
		s.send,
		bundle.WithSize[string](1000),
		bundle.WithTimeout[string](3*time.Second),
		bundle.WithPayloadSize[string](100000),
	)
	s.client = &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	return s
}

func (s *Service) send(UIDs []string) {
	s.logger.Info(
		"推送消息",
		zap.Strings("UIDs", UIDs),
	)
}

func (s *Service) Push(uid string) {
	s.batch.Add(uid)
}

func (s *Service) Start() {
	s.batch.Start()
}

func (s *Service) Close() {
	s.batch.Close()
}
