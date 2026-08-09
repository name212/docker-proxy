package proxy

import (
	"net/http"

	"github.com/name212/docker-proxy/internal/utils/strings"
	"github.com/name212/docker-proxy/internal/utils/request"
)

const requestIDKey = "proxy_request_id"

func getRequestID(r *http.Request) string {
	return request.GetStringFromRequestCtx(r, requestIDKey)
}

func (p *Proxy) getAddRequestIDMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(
				w,
				request.AddStringToRequestCtx(
					r,
					requestIDKey,
					strings.RandString(16),
				),
			)
		})
	}
}
