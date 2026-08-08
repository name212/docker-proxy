package errors

import (
	"fmt"
	"strings"
)

func Join(msg string, errs []string) error {
	return fmt.Errorf("%s:\n\t", strings.Join(errs, "\n\t"))
}
