package proxy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/name212/govalue"
)

type Proxy struct {
	cfg *Config

	startedAddr string
	router      chi.Router
	listener    net.Listener
	server      *http.Server
	stopped     atomic.Bool
	started     atomic.Bool

	logger *Logger

	client *DockerHTTPClient
}

func NewProxy(cfg *Config) (*Proxy, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	p := &Proxy{
		cfg: cfg,
	}

	switch {
	case cfg.UnixSocketPath != "":
		var err error
		p.listener, err = net.Listen("unix", cfg.UnixSocketPath)
		if err != nil {
			return nil, fmt.Errorf("cannot start listener with unix-socket '%s'", cfg.UnixSocketPath)
		}
		p.startedAddr = cfg.UnixSocketPath
	case cfg.BindAddress != "":
		var err error
		p.listener, err = net.Listen("tcp", cfg.BindAddress)
		if err != nil {
			return nil, fmt.Errorf("cannot start listener with bind address '%s'", cfg.BindAddress)
		}
		p.startedAddr = cfg.BindAddress
	default:
		return nil, fmt.Errorf("cannot start proxy. unknown to bind")
	}

	p.logger = newLogger(p.startedAddr)

	client, err := NewDockerHTTPClient(cfg.DockerServer, p.logger)
	if err != nil {
		return nil, p.shutdown("cannot init docker client: %s", err.Error())
	}

	p.client = client

	p.router = chi.NewRouter()

	p.server = &http.Server{
		Handler: p.router,
	}

	return p, nil
}

func (p *Proxy) Start(ctx context.Context) error {
	if p == nil {
		return fmt.Errorf("Proxy is nil")
	}

	if p.started.Load() {
		return fmt.Errorf("already started")
	}

	p.started.Swap(true)

	checkToNil := map[string]any{
		"chi router":     p.router,
		"proxy listener": p.listener,
		"proxy server":   p.server,
		"docker client":  p.client,
		"logger":         p.logger,
	}

	var initErrs []string
	for msg, toCheck := range checkToNil {
		if govalue.IsNil(toCheck) {
			initErrs = append(initErrs, msg)
		}
	}

	if len(initErrs) > 0 {
		return p.shutdown(
			"proxy cannot started, next fields is nil:\n%s",
			strings.Join(initErrs, "\n"),
		)
	}

	p.logger.SetServerCtx(ctx)

	if err := p.initRoutes(ctx); err != nil {
		return p.shutdown("cannot init routes: %w", err)
	}

	go func() {
		<-ctx.Done()
		_ = p.shutdown("")
	}()

	p.logger.Info("Proxy started", p.logger.StringArg("addr", p.startedAddr))

	shutdownF := ""
	var shutdownErr error
	if err := p.server.Serve(p.listener); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			shutdownF = "%w"
			shutdownErr = err
		}
	}

	return p.shutdown(shutdownF, shutdownErr)
}

func (p *Proxy) shutdown(f string, args ...any) error {
	var err error
	if f != "" {
		err = fmt.Errorf(f, args...)
	}

	if !p.stopped.CompareAndSwap(false, true) {
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stopped := true

	if !govalue.IsNil(p.server) {
		if err := p.server.Shutdown(shutdownCtx); err != nil {
			p.logger.Error("Cannot shutdown proxy HTTP-server gracefully", err)
			stopped = false
		} else {
			p.logger.Info("HTTP-sever stopped")
		}
	}

	if !govalue.IsNil(p.listener) {
		if err := p.listener.Close(); err != nil {
			p.logger.Error("Cannot close listener", err)
			stopped = false
		}
	}

	if stopped {
		p.logger.Info("Proxy and listener stopped fully")
	}

	return err
}

func (p *Proxy) initRoutes(ctx context.Context) error {
	p.router.Group(func(r chi.Router) {
		r.Use(
			requestIDMiddleware,
			middleware.Timeout(10*time.Second),
		)

		r.Get("/_healthz", func(w http.ResponseWriter, r *http.Request) {
			p.handleHealthz(ctx, w, r)
		})

		r.Get("/_readyz", func(w http.ResponseWriter, r *http.Request) {
			p.handleReadyz(ctx, w, r)
		})
	})

	p.router.Use(
		requestIDMiddleware,
		p.getAuthMiddleware(),
	)

	p.router.Handle("/", NewProxyHandler(ctx, p))

	return nil
}
