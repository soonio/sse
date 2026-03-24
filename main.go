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

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"pusher/config"
	"pusher/dispatch"
	"pusher/handler"
	"pusher/hub"
	"pusher/limiter"
	"pusher/monitor"
	"pusher/notification"
	"pusher/pool"
)

var configFile = flag.String("f", "config.yml", "the config file")

func main() {
	flag.Parse()

	time.Local, _ = time.LoadLocation("Asia/Shanghai")

	var c config.Config
	config.MustLoad(*configFile, &c)

	// 初始化日志
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

	// 初始化Redis客户端
	cac := redis.NewClient(&redis.Options{
		Addr:            c.Redis.Addr,
		Password:        c.Redis.Password,
		DB:              c.Redis.Database,
		PoolSize:        120,
		MinIdleConns:    64,
		MaxIdleConns:    128,
		ConnMaxIdleTime: 5 * time.Minute,
		ConnMaxLifetime: 10 * time.Minute,
	})

	var err error

	// 创建队列 & 创建消费者组
	err = cac.XGroupCreateMkStream(context.Background(), config.StreamName, config.ConsumerGroup, "$").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		panic(err)
	}

	// 创建消费者
	err = cac.XGroupCreateConsumer(context.Background(), config.StreamName, config.ConsumerGroup, config.ConsumerNamePatten).Err()
	if err != nil {
		panic(err)
	}

	// 初始化连接池
	h := hub.NewHub()
	go h.Start()

	notify := notification.NewNotification(logger)
	notify.Start()

	var bp = pool.NewBufferPool(128)
	var mp = pool.NewMessagePool()

	dispatcher := dispatch.NewDispatcher(
		logger, cac,
		config.StreamName, config.ConsumerGroup, config.ConsumerNamePatten,
		notify, bp, mp, h,
	)
	go dispatcher.Run()

	mon := monitor.NewMonitor(logger, h)
	go mon.Start()

	lim := limiter.NewLimiter()
	go lim.Start()

	srv := &http.Server{Addr: c.Addr, Handler: handler.NewHandler(c.Debug, logger, h, lim, mp)}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Println("http server err:", err)
		}
	}()
	fmt.Printf("ListenOn: %s...\n", c.Addr)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	mon.Stop()
	dispatcher.Close()
	notify.Close()
	h.Clear()
	lim.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error(err.Error())
	}
}
