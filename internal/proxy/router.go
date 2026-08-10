package proxy

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (p *Proxy) initRoutes(ctx context.Context) {
	p.router.Use(
		p.getAddRequestIDMiddleware(),
	)

	p.router.Group(func(r chi.Router) {
		r.Use(
			middleware.Timeout(10 * time.Second),
		)

		r.Get("/_healthz", func(w http.ResponseWriter, r *http.Request) {
			p.handleHealthz(w, r)
		})

		r.Get("/_readyz", func(w http.ResponseWriter, r *http.Request) {
			p.handleReadyz(w, r)
		})
	})

	p.router.Group(func(r chi.Router) {
		r.Use(
			p.getAuthMiddleware(),
		)

		r.Handle("/*", NewProxyHandler(ctx, p))
	})

	p.router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		writeInternalServerErrorResponse("docker-proxy: router not handle request", w, r, p.logger)
	})
}
