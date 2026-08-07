package app

import (
	"context"

	flag "github.com/spf13/pflag"

	"github.com/name212/docker-proxy/internal/proxy"
)

func GetProxyConfigFromArgs(ctx context.Context) (*proxy.Config, error) {
	if err := setLogger(&loggerConfig{}); err != nil {
		return nil, err
	}

	serverConfigPath := flag.StringP(
		"proxy-config-path",
		"c",
		"",
		`YAML proxy config. Another server options will skip if passed
  Format:
    unixSocketPath: path to create unix-socket for handle requests. If passed bindAddress should not set
    bindAddress: address to bind proxy. If passed unixSocketPath should not set
	dockerAddress: address or unix-socket path to docker server
    usersConfigPath: path to users config file. Required. Should have owner root:root and 600 permission
	Format:
	  users: list of users
	  - name: name or description of user
	  - token: token for auth user
	  - roles: list of available roles:
	    - admin - full access to docker API
	logLevel: level of logger, default DEBUG
	  if passed via config some logs can be printed with debug level before full init
	  Can be:
	  - DEBUG
	  - INFO
	  - WARN
	  - ERROR
	logFormat: format log messages, Default json
	   if passed via config some logs can be printed with text format level before full init
	   Can be:
	   - text
	   - json
`)

	appConfig := &Config{}
	logConf := &loggerConfig{}

	flag.StringVar(&appConfig.UnixSocketPath, "server-unix-socket-path", "", "same unixSocketPath in proxy config")
	flag.StringVar(&appConfig.BindAddress, "server-bind-address", "", "same bindAddress in proxy config")
	flag.StringVar(&appConfig.DockerAddress, "docker-address", "", "same dockerAddress in proxy config")
	flag.StringVar(&appConfig.UsersConfigPath, "server-users-config-path", "", "same usersConfigPath in proxy config")
	flag.StringVar(&logConf.level, "log-level", defaultLogLevel, "same logLevel in proxy config")
	flag.StringVar(&appConfig.LogFormat, "log-format", defaultFormatJSON, "same logFormat in proxy config")

	flag.Parse()

	if serverConfigPath != nil && *serverConfigPath != "" {
		var err error
		appConfig, err = ReadAppConfigFromFile(ctx, *serverConfigPath)
		if err != nil {
			return nil, err
		}

		logConf = &loggerConfig{
			level:  appConfig.LogLevel,
			format: appConfig.LogFormat,
		}
	}

	if err := initLogger(logConf); err != nil {
		return nil, err
	}

	return GetProxyConfig(appConfig)
}
