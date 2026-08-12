// Copyright 2026
// license that can be found in the LICENSE file.

package permissions

import (
	"fmt"
	"os"
	"os/user"
	"slices"
	"strings"
)

var (
	AllowSkipEnv = "false"
	trueVars     = []string{"true", "1"}
)

func isCheckPermissions() bool {
	if AllowSkipEnv != "true" {
		return true
	}

	skipEnvVal := os.Getenv("SKIP_CHECK_PERMISSIONS")
	skipEnvVal = strings.ToLower(skipEnvVal)

	return !slices.Contains(trueVars, skipEnvVal)
}

func IsRunAsRoot() error {
	if !isCheckPermissions() {
		return nil
	}

	u, err := user.Current()
	if err != nil {
		return fmt.Errorf("cannot get user: %w", err)
	}

	if u.Uid != "0" {
		return fmt.Errorf("not run as root. current user %s (%s)", u.Name, u.Uid)
	}

	return nil
}
