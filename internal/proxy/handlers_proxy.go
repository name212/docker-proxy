package proxy

import (
	"context"
	"net/http"
)

var cannotSendRequestErr = []byte("Cannot send request or read response to/from docker")

type ProxyHandler struct {
	serverCtx context.Context
	p         *Proxy
}

func NewProxyHandler(serverCtx context.Context, p *Proxy) *ProxyHandler {
	return &ProxyHandler{
		p:         p,
		serverCtx: serverCtx,
	}
}

func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := h.p.client.Send(r.Context(), w, r); err != nil {
		writeInternalServerErrorResponse(cannotSendRequestErr, w, r, h.p.logger)
	}
}
