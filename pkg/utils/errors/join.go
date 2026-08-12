// Copyright 2026
// license that can be found in the LICENSE file.

package errors

import (
	"fmt"
	"strings"
)

func Join(msg string, errs []string) error {
	return fmt.Errorf("%s:\n\t", strings.Join(errs, "\n\t"))
}
