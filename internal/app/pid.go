// Copyright 2026
// license that can be found in the LICENSE file.

package app

import (
	"fmt"
	"os"
)

func writePIDFile(pidFile string) error {
	if pidFile == "" {
		return nil
	}

	content := fmt.Appendf(nil, "%d", os.Getpid())

	if err := os.WriteFile(pidFile, content, 0o644); err != nil {
		return fmt.Errorf("cannot write pid '%s' to file %s: %w", content, pidFile, err)
	}

	return nil
}