package main

import (
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/gzlj/http-demo/pkg/prober"
	httpserver "github.com/gzlj/http-demo/pkg/server/http"
	"github.com/oklog/run"
	"github.com/pkg/errors"
	"os"
	"os/signal"
	"syscall"
	"time"
)


const (
	logFormatLogfmt                     = "logfmt"
	logFormatJson                       = "json"
)

func main() {
	var logFormat string = "logfmt"
	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))
	if logFormat == logFormatJson {
		logger = log.NewJSONLogger(log.NewSyncWriter(os.Stdout))
	}
	logger = log.With(logger, "ts", log.DefaultTimestamp)
	logger = log.With(logger, "caller", log.DefaultCaller)

	httpProbe := prober.NewHTTP()
	probers := prober.Combine(
		httpProbe,
	)

	//返回http服务结构体实例，它封装了原生http服务
	srv := httpserver.New(
		logger,
		"test-http",
		httpProbe,
		httpserver.WithListen(":80"),
		httpserver.WithGracePeriod(time.Duration(5)),
	)

	var g run.Group
	//启动http服务
	g.Add(
	//此方法是actor的execute()方法
	func() error {
		probers.Healthy()
		return srv.ListenAndServe()
	},
	//此方法是actor的interrupt()方法
	func(err error) {
		probers.NotReady(err)
		defer probers.NotHealthy(err)
		//平滑关闭http服务
		srv.Shutdown(err)
	})

	//监听来自操作系统的杀死信号
	{
		cancel := make(chan struct{})
		g.Add(func() error {
			return interrupt(logger, cancel)
		}, func(error) {
			close(cancel)
		})
	}

	//Run方法中，先遍历所有actor执行actor的execute()方法，一旦有一个actor返回error接口（值可能是nil），则遍历actor调用其interrupt()方法（入参都是这个error接口 ）
	//interrupt()中往往记录日志以及平滑关闭actor
	//Run方法最后部分是等待所有剩余actor执行退出
	//Run方法返回的是第一个actor返回的error接口
	if err := g.Run(); err != nil {
		level.Error(logger).Log("err", fmt.Sprintf("%+v", errors.Wrapf(err, "command run failed")))
		os.Exit(1)
	}
	level.Info(logger).Log("msg", "exiting")
}

func interrupt(logger log.Logger, cancel <-chan struct{}) error {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	//接收到杀死信号或结束信号时，进行方法返回
	select {
	case s := <-c:
		level.Info(logger).Log("msg", "caught signal. Beginng to exit.", "signal", s)
		return nil
	case <-cancel:
		return errors.New("canceled")
	}
}
