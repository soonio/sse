package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"pusher/app"
	"pusher/config"
)

var configFile = flag.String("f", "config.yml", "the config file")

func main() {
	flag.Parse()

	time.Local, _ = time.LoadLocation("Asia/Shanghai")

	var c config.Config
	config.MustLoad(*configFile, &c)

	logger := zap.New(
		zapcore.NewCore(
			zapcore.NewJSONEncoder(zapcore.EncoderConfig{
				TimeKey:       "time",
				LevelKey:      "level",
				NameKey:       "logger",
				CallerKey:     "caller",
				FunctionKey:   zapcore.OmitKey,
				MessageKey:    "msg",
				StacktraceKey: "stacktrace",
				LineEnding:    zapcore.DefaultLineEnding,
				EncodeLevel:   zapcore.LowercaseLevelEncoder,
				EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
					enc.AppendString(t.Format("2006-01-02 15:04:05"))
				},
				EncodeDuration: zapcore.SecondsDurationEncoder,
				EncodeCaller:   zapcore.ShortCallerEncoder,
			}),
			zapcore.Lock(os.Stdout),
			zap.InfoLevel,
		),
		zap.AddCaller(),
	)

	a, err := app.InitApp(&c, logger)
	if err != nil {
		panic(err)
	}

	// 初始化 Redis Stream 消费者组
	ctx := context.Background()
	rdbClient := a.PushSvc.RedisClient()
	if err := rdbClient.XGroupCreateMkStream(ctx, config.StreamName, config.ConsumerGroup, "$").Err(); err != nil {
		if !strings.Contains(err.Error(), "BUSYGROUP") {
			panic(err)
		}
	}
	if err := rdbClient.XGroupCreateConsumer(ctx, config.StreamName, config.ConsumerGroup, config.ConsumerNamePatten).Err(); err != nil {
		panic(err)
	}

	// 启动各组件
	go a.Hub.Start()
	a.NotifySvc.Start()
	go a.PushSvc.Run()
	go a.Monitor.Start()
	go a.Limiter.Start()

	go func() {
		if err := a.HTTPSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Println("http server err:", err)
		}
	}()
	fmt.Printf("ListenOn: %s...\n", c.Addr)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	a.Monitor.Stop()
	a.PushSvc.Close()
	a.NotifySvc.Close()
	a.Hub.Clear()
	a.Limiter.Close()

	shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := a.HTTPSrv.Shutdown(shutCtx); err != nil {
		logger.Error(err.Error())
	}
}
