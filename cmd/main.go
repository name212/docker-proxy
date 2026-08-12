// Copyright 2026
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/name212/docker-proxy/internal/app"
	"github.com/name212/docker-proxy/internal/proxy"
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	proxyConf, err := app.GetProxyConfigFromArgs(ctx)
	if err != nil {
		slog.Error("cannot get proxy config", slog.String("err", err.Error()))
		return err
	}

	proxyServer, err := proxy.NewProxy(proxyConf)
	if err != nil {
		slog.Error("cannot init proxy", slog.String("err", err.Error()))
		return err
	}

	if err := proxyServer.Start(ctx); err != nil {
		slog.Error("proxy stopped with error", slog.String("err", err.Error()))
		return err
	}

	return nil
}
