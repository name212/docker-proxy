package proxy

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/name212/docker-proxy/pkg/auth"
	"github.com/name212/docker-proxy/pkg/auth/users"
	"github.com/name212/docker-proxy/pkg/utils/request"
)

const (
	authTokenHeader    = "X-Auth-Token"
	requestUserNameKey = "user_name"
)

var (
	emptyTokenErrMsg = fmt.Sprintf("token header %s not passed or empty", authTokenHeader)
)

func getUserNameForRequest(r *http.Request) string {
	return request.GetStringFromRequestCtx(r, requestUserNameKey)
}

func (p *Proxy) getAuthMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get(authTokenHeader)
			if token == "" {
				writeNotAuthorizedErr(w, r, emptyTokenErrMsg, p.logger)
				return
			}

			allowRes, err := p.cfg.Authorizer.Allow(
				r.Context(),
				users.Token(token),
				r.URL,
				r.Method,
			)

			if err != nil {
				if errors.Is(err, auth.ErrUnauthorized) {
					errMsg := fmt.Sprintf("not authorized for request: %s", err.Error())
					writeNotAuthorizedErr(w, r, errMsg, p.logger)
					return
				}

				errMsg := fmt.Sprintf(
					"got unexpected error while checking allow: %s",
					err.Error(),
				)

				writeInternalServerErrorResponse(errMsg, w, r, p.logger)
				return
			}

			r.Header.Del(authTokenHeader)
			r = request.AddStringToRequestCtx(r, requestUserNameKey, allowRes.User)

			p.logger.Request(
				r,
				InfoCtx,
				"allow request for user",
				// user name will get from ctx
				p.logger.StringArg("allow_by", allowRes.String()),
			)

			next.ServeHTTP(w, r)
		})
	}
}

func writeNotAuthorizedErr(w http.ResponseWriter, r *http.Request, msg string, logger *Logger) {
	logger.Request(r, WarnCtx, msg)

	w.WriteHeader(http.StatusUnauthorized)

	if _, err := w.Write([]byte(msg)); err != nil {
		logger.Error("cannot write unauthorized response", auth.ErrUnauthorized, r)
	}
}
