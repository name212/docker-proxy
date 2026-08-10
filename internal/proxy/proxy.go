package proxy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
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

	postCloseListener func(logger *Logger)

	logger *Logger

	client *DockerHTTPClient
}

func NewProxy(cfg *Config) (*Proxy, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	p := &Proxy{
		cfg:               cfg,
		logger:            newLogger(cfg.DockerServer),
		postCloseListener: func(logger *Logger) {},
	}

	var startListener func() error

	startUnixProxy := func() error {
		socketPath := cfg.UnixSocketPath

		var err error

		p.postCloseListener = func(logger *Logger) {
			err := os.Remove(socketPath)

			if err == nil {
				return
			}

			if !errors.Is(err, os.ErrNotExist) {
				msg := fmt.Sprintf("Cannot remove socket file '%s'", socketPath)
				logger.Error(msg, err)
			}
		}

		p.listener, err = net.Listen("unix", socketPath)
		if err != nil {
			return fmt.Errorf("cannot start listener with unix-socket '%s': %w", socketPath, err)
		}

		if err := os.Chmod(socketPath, 0o777); err != nil {
			return fmt.Errorf("cannot chmod to 777 socket file '%s': %w", socketPath, err)
		}

		p.startedAddr = socketPath

		return nil
	}

	startPortProxy := func() error {
		var err error
		p.listener, err = net.Listen("tcp", cfg.BindAddress)
		if err != nil {
			return fmt.Errorf("cannot start listener with bind address '%s': %w", cfg.BindAddress, err)
		}
		p.startedAddr = cfg.BindAddress
		return nil
	}

	switch {
	case cfg.UnixSocketPath != "":
		startListener = startUnixProxy
	case cfg.BindAddress != "":
		startListener = startPortProxy
	default:
		return nil, fmt.Errorf("cannot start proxy. unknown to bind")
	}

	if err := startListener(); err != nil {
		return nil, p.shutdown("cannot start listener: %s", err.Error())
	}

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
		"authorizer":     p.cfg.Authorizer,
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

	p.initRoutes(ctx)

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
			if !errors.Is(err, net.ErrClosed) {
				p.logger.Error("Cannot close listener", err)
				stopped = false
			}
		}
	}

	if stopped {
		if p.postCloseListener != nil {
			p.postCloseListener(p.logger)
		}
		p.logger.Info("Proxy and listener stopped fully")
	}

	return err
}
