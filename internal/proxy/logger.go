// Copyright 2026
// license that can be found in the LICENSE file.

package proxy

import (
	"context"
	"log/slog"
	"net/http"
)

type (
	loggerFuncCtx = func(ctx context.Context, msg string, args ...any)
)

var (
	DebugCtx loggerFuncCtx = slog.DebugContext
	InfoCtx  loggerFuncCtx = slog.InfoContext
	WarnCtx  loggerFuncCtx = slog.WarnContext
	ErrorCtx loggerFuncCtx = slog.ErrorContext
)

type Logger struct {
	server    string
	serverCtx context.Context
}

func newLogger(server string) *Logger {
	return &Logger{
		server:    server,
		serverCtx: context.Background(),
	}
}

func (l *Logger) SetServerCtx(serverCtx context.Context) {
	l.serverCtx = serverCtx
}

func (l *Logger) Request(r *http.Request, lFunc loggerFuncCtx, msg string, args ...any) {
	lFunc(l.serverCtx, msg, l.getLogArgs(r, args)...)
}

func (l *Logger) Response(r *http.Response, lFunc loggerFuncCtx, msg string, args ...any) {
	defaults := []any{
		slog.String("status", r.Status),
		slog.Int("status_code", r.StatusCode),
	}

	all := make([]any, 0, len(defaults)+len(args))

	all = append(all, args...)

	lFunc(l.serverCtx, msg, l.getLogArgs(r.Request, all)...)
}

func (l *Logger) StringArg(k, v string) any {
	return slog.String(k, v)
}

func (l *Logger) Info(msg string, args ...any) {
	slog.Info(msg, args...)
}

func (l *Logger) Error(msg string, err error, r ...*http.Request) {
	var all []any
	if err != nil {
		all = append(all, slog.String("err", err.Error()))
	}

	if len(r) > 0 {
		all = l.getLogArgs(r[0], all)
	}

	slog.Error(msg, all...)
}

func (l *Logger) getLogArgs(r *http.Request, args []any) []any {
	defaults := []any{
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.String("full_url", r.URL.String()),
		slog.String("via", l.server),
		slog.String(requestIDKey, getRequestID(r)),
		slog.String(requestUserNameKey, getUserNameForRequest(r)),
	}

	res := make([]any, 0, len(defaults)+len(args))
	res = append(res, defaults...)

	return append(res, args...)
}
