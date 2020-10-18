package http

import (
	"context"
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/gzlj/http-demo/pkg/prober"
	"github.com/pkg/errors"
	"net/http"
)

// A Server defines parameters for serve HTTP requests, a wrapper around http.Server.
type Server struct {
	logger log.Logger
	//comp   component.Component
	prober *prober.HTTPProbe

	mux *http.ServeMux
	srv *http.Server

	opts options
}

// constructor
func New(logger log.Logger, name string, prober *prober.HTTPProbe, opts ...Option) *Server {
	options := options{}
	for _, o := range opts {
		o.apply(&options)
	}
	mux := http.NewServeMux()
	registerProbes(mux, prober, logger)
	registerGetClient(mux)

	return &Server{
		logger: log.With(logger, "service", "http/server", "component", name),
		prober: prober,
		mux:    mux,
		srv:    &http.Server{Addr: options.listen, Handler: mux},
		opts:   options,
	}
}

func registerProbes(mux *http.ServeMux, p *prober.HTTPProbe, logger log.Logger) {
	if p != nil {
		mux.Handle("/-/healthy", p.HealthyHandler(logger))
		mux.Handle("/-/ready", p.ReadyHandler(logger))
	}
}

func registerGetClient(mux *http.ServeMux) {
	mux.Handle("/ip", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		str := fmt.Sprintf("Recieve from server: client is: %s", r.RemoteAddr)
		w.Write([]byte(str))
	}))
}


// ListenAndServe listens on the TCP network address and handles requests on incoming connections.
func (s *Server) ListenAndServe() error {
	level.Info(s.logger).Log("msg", "listening for requests and metrics", "address", s.opts.listen)
	return errors.Wrap(s.srv.ListenAndServe(), "serve HTTP and metrics")
}

// Shutdown gracefully shuts down the server by waiting,
// for specified amount of time (by gracePeriod) for connections to return to idle and then shut down.
func (s *Server) Shutdown(err error) {
	level.Info(s.logger).Log("msg", "internal server is shutting down", "err", err)
	if err == http.ErrServerClosed {
		level.Warn(s.logger).Log("msg", "internal server closed unexpectedly")
		return
	}

	if s.opts.gracePeriod == 0 {
		s.srv.Close()
		level.Info(s.logger).Log("msg", "internal server is shutdown", "err", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.opts.gracePeriod)
	defer cancel()

	if err := s.srv.Shutdown(ctx); err != nil {
		level.Error(s.logger).Log("msg", "internal server shut down failed", "err", err)
		return
	}
	level.Info(s.logger).Log("msg", "internal server is shutdown gracefully", "err", err)
}

// Handle registers the handler for the given pattern.
func (s *Server) Handle(pattern string, handler http.Handler) {
	s.mux.Handle(pattern, handler)
}
