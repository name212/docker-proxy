package proxy

import (
	"context"

	"github.com/go-chi/chi/v5/middleware"
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

var requestIDMiddleware = middleware.WithValue(RequestIDKey, rand.String(16))
