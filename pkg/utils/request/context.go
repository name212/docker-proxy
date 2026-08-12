// Copyright 2026
// license that can be found in the LICENSE file.

package request

import (
	"context"
	"net/http"

	"github.com/name212/govalue"
)

type RequestKey string

func AddStringToRequestCtx(r *http.Request, k RequestKey, v string) *http.Request {
	ctx := context.WithValue(r.Context(), k, v)
	return r.WithContext(ctx)
}

func GetStringFromRequestCtx(r *http.Request, k string) string {
	res := "n/a"
	id := r.Context().Value(RequestKey(k))
	if !govalue.IsNil(id) {
		idStr, ok := id.(string)
		if ok {
			res = idStr
		}
	}

	return res
}
