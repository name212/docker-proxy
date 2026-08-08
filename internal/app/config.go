package app

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"syscall"

	yaml "github.com/goccy/go-yaml"

	server "github.com/name212/docker-proxy/internal/proxy"
)

const DefaultDockerUNIXSocket = "/run/docker.sock"

type Config struct {
	UnixSocketPath  string `yaml:"unixSocketPath"`
	BindAddress     string `yaml:"bindAddress"`
	UsersConfigPath string `yaml:"usersConfigPath"`
	DockerAddress   string `yaml:"dockerAddress"`

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

	return checkFile(c.UsersConfigPath, "users config")
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

	if err := checkFile(path, "proxy app config"); err != nil {
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

func GetProxyConfig(appConfig *Config) (*server.Config, error) {
	if err := appConfig.prepareAndValidate(); err != nil {
		return nil, fmt.Errorf("cannot validate app proxy config: %w", err)
	}

	usersCfgContent, err := os.ReadFile(appConfig.UsersConfigPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read users config '%s': %w", appConfig.UsersConfigPath, err)
	}

	usersCfg := server.UsersConfig{}
	if err := yaml.Unmarshal(usersCfgContent, &usersCfg); err != nil {
		return nil, fmt.Errorf("cannot unmarshal users config '%s': %w", appConfig.UsersConfigPath, err)
	}

	usersMap, err := usersCfg.ExtractUsersMap()
	if err != nil {
		return nil, err
	}

	return &server.Config{
		DockerServer:   appConfig.DockerAddress,
		UnixSocketPath: appConfig.UnixSocketPath,
		BindAddress:    appConfig.BindAddress,
		Users:          usersMap,
	}, nil
}

func checkFile(path, errPrefix string) error {
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
