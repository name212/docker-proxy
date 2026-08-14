// Copyright 2026
// license that can be found in the LICENSE file.

package app

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	yaml "github.com/goccy/go-yaml"

	initauth "github.com/name212/docker-proxy/internal/auth"
	"github.com/name212/docker-proxy/internal/proxy"
	"github.com/name212/docker-proxy/pkg/auth"
	"github.com/name212/docker-proxy/pkg/utils/permissions"
)

const DefaultDockerUNIXSocket = "/run/docker.sock"

type Config struct {
	UnixSocketPath  string `yaml:"unixSocketPath"`
	BindAddress     string `yaml:"bindAddress"`
	UsersConfigPath string `yaml:"usersConfigPath"`
	DockerAddress   string `yaml:"dockerAddress"`
	PIDFIle         string `yaml:"pidFile"`

	LogLevel  string `yaml:"logLevel"`
	LogFormat string `yaml:"logFormat"`
}

func (c *Config) prepareAndValidate() error {
	if c == nil {
		return fmt.Errorf("nil config passed to validate")
	}

	if c.DockerAddress == "" {
		c.DockerAddress = DefaultDockerUNIXSocket
	}

	return permissions.FileIsRootAccessOnly(c.UsersConfigPath, "users config")
}

func ReadAppConfig(r io.Reader) (*Config, error) {
	res := Config{}

	content, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(content, &res); err != nil {
		return nil, fmt.Errorf("cannot unmarshal yaml app config: %w", err)
	}

	return &res, nil
}

func ReadAppConfigFromFile(ctx context.Context, path string) (*Config, error) {
	slog.DebugContext(ctx, "Got proxy config file", slog.String("path", path))

	if err := permissions.FileIsRootAccessOnly(path, "proxy app config"); err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read proxy config file '%s': %w", path, err)
	}

	defer func() {
		if err := f.Close(); err != nil {
			slog.ErrorContext(
				ctx,
				"Cannot close proxy config file",
				slog.String("path", path),
				slog.String("err", err.Error()),
			)
		}
	}()

	return ReadAppConfig(f)
}

func GetProxyConfig(ctx context.Context, appConfig *Config) (*proxy.Config, *auth.Authorizer, error) {
	if err := appConfig.prepareAndValidate(); err != nil {
		return nil, nil, fmt.Errorf("cannot validate app proxy config: %w", err)
	}

	usersCfgContent, err := os.ReadFile(appConfig.UsersConfigPath)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot read users config '%s': %w", appConfig.UsersConfigPath, err)
	}

	usersCfg := auth.UsersConfig{}
	if err := yaml.Unmarshal(usersCfgContent, &usersCfg); err != nil {
		return nil, nil, fmt.Errorf("cannot unmarshal users config '%s': %w", appConfig.UsersConfigPath, err)
	}

	authorizer, err := initauth.CreateAuthorizer(ctx, &usersCfg)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot create authorizer: %w", err)
	}

	return &proxy.Config{
		DockerServer:   appConfig.DockerAddress,
		UnixSocketPath: appConfig.UnixSocketPath,
		BindAddress:    appConfig.BindAddress,
	}, authorizer, nil
}
