// Copyright 2026
// license that can be found in the LICENSE file.

package request

import (
	"fmt"
	"net/http"
)

func RequestStr(r *http.Request) string {
	if r == nil {
		return "'UNKNOWN'"
	}

	path := "UNKNOWN PATH"
	if r.URL != nil {
		path = r.URL.Path
	}

	return fmt.Sprintf("'%s %s'", r.Method, path)
}
