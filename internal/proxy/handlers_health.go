// Copyright 2026
// license that can be found in the LICENSE file.

package proxy

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

const dockerPingPath = "/v1.55/_ping"

var (
	okMsg                = []byte("OK")
	internalServerErrMsg = "Internal server error"

	dockerPingMethod = http.MethodHead
)

func (p *Proxy) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeOKResponse(w, r, p.logger)
}

func (p *Proxy) handleReadyz(w http.ResponseWriter, r *http.Request) {
	pingContext, cancel := context.WithTimeout(r.Context(), 4*time.Second)
	defer cancel()

	pingRequest, err := http.NewRequestWithContext(pingContext, dockerPingMethod, dockerPingPath, nil)
	if err != nil {
		p.logger.Error(
			fmt.Sprintf("Cannot build ping request: %s %s", dockerPingMethod, dockerPingPath),
			err,
			r,
		)

		writeInternalServerErrorResponse(internalServerErrMsg, w, r, p.logger)
		return
	}

	if err := p.client.SendAndGetOnlyStatus(pingContext, pingRequest); err != nil {
		errMsg := fmt.Sprintf("docker ping failed: %s", err.Error())
		writeInternalServerErrorResponse(errMsg, w, r, p.logger)
		return
	}

	writeOKResponse(w, r, p.logger)
}

func writeInternalServerErrorResponse(responseMsg string, w http.ResponseWriter, r *http.Request, logger *Logger) {
	logger.Error(responseMsg, fmt.Errorf("%s", responseMsg), r)

	w.WriteHeader(http.StatusInternalServerError)

	if _, err := w.Write([]byte(responseMsg)); err != nil {
		logger.Error("Cannot write internal error response", err, r)
	}
}

func writeOKResponse(w http.ResponseWriter, r *http.Request, logger *Logger) {
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(okMsg); err != nil {
		logger.Error("Cannot write ok response", err, r)
	}
}
