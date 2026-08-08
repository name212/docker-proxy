package request

import (
	"context"
	"net/http"

	"github.com/name212/govalue"
)

func AddStringToRequestCtx(r *http.Request, k, v string) *http.Request {
	ctx := context.WithValue(r.Context(), k, v)
	return r.WithContext(ctx)
}

func GetStringFromRequestCtx(r *http.Request, k string) string {
	res := "n/a"
	id := r.Context().Value(k)
	if !govalue.IsNil(id) {
		idStr, ok := id.(string)
		if ok {
			res = idStr
		}
	}

	return res
}
