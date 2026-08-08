package app

import (
	"context"
	"fmt"
	"os"
	"strings"

	flag "github.com/spf13/pflag"

	"github.com/name212/docker-proxy/internal/proxy"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, `

For use docker-proxy for docker-cli you should:
  Pass env DOCKER_HOST with address to proxy (see unixSocketPath/bindAddress params)
    If you use unixSocketPath, DOCKER_HOST should contains prefix unix:// like
      unix:///run/docker-proxy.socket
    If you use bindAddress, DOCKER_HOST should contains prefix tcp:// like
      tcp://127.0.0.1:8081
  Pass env DOCKER_CUSTOM_HEADERS with X-Auth-Token header. Header value is user token from usersConfigPath
Example:
  export DOCKER_CUSTOM_HEADERS="X-Auth-Token=EXAMPLE-T0Ken-1111"
  export DOCKER_HOST="from unixSocketPath/bindAddress params with required proto prefix"
  docker image ls
`)
}

func GetProxyConfigFromArgs(ctx context.Context) (*proxy.Config, error) {
	if err := setLogger(&loggerConfig{}); err != nil {
		return nil, err
	}

	flag.Usage = usage

	rolesList := proxy.RolesDescriptions()
	rolesSeparator := "	    - "
	rolesListStr := strings.Join(rolesList, "\n"+rolesSeparator)
	rolesListStr = fmt.Sprintf("%s%s", rolesSeparator, rolesListStr)

	serverConfigPath := flag.StringP(
		"proxy-config-path",
		"c",
		"",
		fmt.Sprintf(`YAML proxy config. Another server options will skip if passed
  Format:
    unixSocketPath: path to create unix-socket for handle requests. If passed bindAddress should not set
    bindAddress: address to bind proxy. If passed unixSocketPath should not set
	dockerAddress: address or unix-socket path to docker server.
	  By default: /run/docker.sock
    usersConfigPath: path to users config file. Required. Should have owner root:root and 600 permission
	Format:
	  users: list of users
	  - name: name or description of user
	  - token: token for auth user (token should be len >= 16)
	  - roles: list of available roles:
%s
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
`, rolesListStr))

	appConfig := &Config{}
	argsAppConfig := appConfig
	logConf := &loggerConfig{}

	flag.StringVar(&appConfig.UnixSocketPath, "server-unix-socket-path", "", "same unixSocketPath in proxy config")
	flag.StringVar(&appConfig.BindAddress, "server-bind-address", "", "same bindAddress in proxy config")
	flag.StringVar(&appConfig.DockerAddress, "docker-address", DefaultDockerUNIXSocket, "same dockerAddress in proxy config")
	flag.StringVar(&appConfig.UsersConfigPath, "server-users-config-path", "", "same usersConfigPath in proxy config")
	flag.StringVar(&logConf.level, "log-level", defaultLogLevel, "same logLevel in proxy config")
	flag.StringVar(&appConfig.LogFormat, "log-format", defaultFormatJSON, "same logFormat in proxy config")

	flag.Parse()

	if err := IsRunAsRoot(); err != nil {
		return nil, err
	}

	if serverConfigPath != nil && *serverConfigPath != "" {
		var err error
		appConfig, err = ReadAppConfigFromFile(ctx, *serverConfigPath)
		if err != nil {
			return nil, err
		}

		logLevel := appConfig.LogLevel
		logFormat := appConfig.LogFormat

		if logLevel == "" {
			logLevel = argsAppConfig.LogLevel
		}

		if logFormat == "" {
			logFormat = argsAppConfig.LogFormat
		}

		logConf = &loggerConfig{
			level:  logLevel,
			format: logFormat,
		}
	}

	if err := initLogger(logConf); err != nil {
		return nil, err
	}

	return GetProxyConfig(appConfig)
}
