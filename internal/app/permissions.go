package app

import (
	"fmt"
	"os"
	"os/user"
	"slices"
	"strings"
)


var trueVars = []string{"true", "1"} 

func isCheckPermissions() bool {
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