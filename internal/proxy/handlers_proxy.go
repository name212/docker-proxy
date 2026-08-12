// Copyright 2026
// license that can be found in the LICENSE file.

package proxy

import (
	"context"
	"net/http"
)

var cannotSendRequestErrMsg = "Cannot send request or read response to/from docker"

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
		h.p.logger.Error(cannotSendRequestErrMsg, err, r)
		writeInternalServerErrorResponse(cannotSendRequestErrMsg, w, r, h.p.logger)
	}
}
