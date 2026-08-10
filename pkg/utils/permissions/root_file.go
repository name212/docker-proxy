package permissions

import (
	"fmt"
	"os"
	"syscall"
)

func FileIsRootAccessOnly(path, errPrefix string) error {
	retErr := func(f string, args ...any) error {
		pref := fmt.Sprintf("%s path '%s' ", errPrefix, path)
		return fmt.Errorf(pref+f, args...)
	}

	if path == "" {
		return retErr("not passed")
	}

	usersStat, err := os.Stat(path)
	if err != nil {
		return retErr("not found or not readable: %w", err)
	}

	if usersStat.IsDir() {
		return retErr("is dir")
	}

	if usersStat.Size() == 0 {
		return retErr("is empty")
	}

	usersUnixStat, ok := usersStat.Sys().(*syscall.Stat_t)
	if !ok {
		return retErr("not a Unix-like file system")
	}

	if isCheckPermissions() {
		if usersUnixStat.Uid != 0 || usersUnixStat.Gid != 0 {
			return retErr(
				"have incorrect owner (%d:%d) should be root",
				usersUnixStat.Uid,
				usersUnixStat.Gid,
			)
		}

		if perm := usersStat.Mode().Perm(); perm != 0o600 {
			return retErr("have incorrect permission should be 600")
		}
	}

	return nil
}
