package proxy

import (
	"errors"
	"fmt"
	"net/http"
)

const authTokenHeader = "X-Auth-Token"

var (
	emptyTokenErrMsg   = fmt.Sprintf("token header %s not passed or empty", authTokenHeader)
	userNotFoundErrMsg = "user not found by passed token"
)

func (p *Proxy) getAuthMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get(authTokenHeader)
			if token == "" {
				writeNotAuthorizedErr(w, r, emptyTokenErrMsg, p.logger)
				return
			}

			user, ok := p.cfg.Users[Token(token)]
			if !ok {
				writeNotAuthorizedErr(w, r, userNotFoundErrMsg, p.logger)
				return
			}

			userName := user.Name

			allowBy, err := IsAllow(user, r.URL, r.Method)

			if err != nil {
				if errors.Is(err, UnauthorizedErr) {
					errMsg := fmt.Sprintf("user '%s' not authorized for request", user)
					writeNotAuthorizedErr(w, r, errMsg, p.logger)
					return
				}

				errMsg := fmt.Appendf(
					nil,
					"got unexpected error while checking allow for user '%s': '%s'",
					userName,
					err.Error(),
				)

				writeInternalServerErrorResponse(errMsg, w, r, p.logger)
				return
			}

			p.logger.Request(
				r,
				InfoCtx,
				"allow request for user",
				p.logger.StringArg("user", userName),
				p.logger.StringArg("allow_by", allowBy),
			)

			next.ServeHTTP(w, r)
		})
	}
}

func writeNotAuthorizedErr(w http.ResponseWriter, r *http.Request, msg string, logger *Logger) {
	logger.Request(r, WarnCtx, msg)

	w.WriteHeader(http.StatusUnauthorized)

	if _, err := w.Write([]byte(msg)); err != nil {
		logger.Error("cannot write unauthorized response", UnauthorizedErr, r)
	}
}
