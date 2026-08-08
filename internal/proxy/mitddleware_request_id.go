package proxy

import (
	"context"
	"net/http"

	"github.com/name212/govalue"

	"github.com/name212/docker-proxy/internal/utils/rand"
)

const RequestIDKey = "proxy_request_id"

func getRequestID(ctx context.Context) string {
	res := "n/a"
	id := ctx.Value(RequestIDKey)
	if !govalue.IsNil(id) {
		idStr, ok := id.(string)
		if ok {
			res = idStr
		}
	}

	return res
}

func (p *Proxy) getAddRequestIDMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctxWithRequestID := context.WithValue(r.Context(), RequestIDKey, rand.String(16))
			next.ServeHTTP(w, r.WithContext(ctxWithRequestID))
		})
	}
}
