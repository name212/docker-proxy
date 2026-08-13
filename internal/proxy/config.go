// Copyright 2026
// license that can be found in the LICENSE file.

package proxy

import (
	"github.com/name212/docker-proxy/pkg/utils/errors"
)

type Config struct {
	UnixSocketPath string
	BindAddress    string
	DockerServer   string
}

func (c *Config) Validate() error {
	var errs []string

	if c.DockerServer == "" {
		errs = append(errs, "docker target address should be passed")
	}

	if c.UnixSocketPath == "" && c.BindAddress == "" {
		errs = append(errs, "bind address or unix socket should be passed")
	}

	if c.UnixSocketPath != "" && c.BindAddress != "" {
		errs = append(errs, "bind address and unix socket path should not be passed both")
	}

	if len(errs) > 0 {
		return errors.Join("Server config invalid", errs)
	}

	return nil
}
