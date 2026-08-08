package proxy

import (
	"fmt"

	"github.com/name212/docker-proxy/internal/utils/errors"
)

type UsersConfig struct {
	Users []*User `yaml:"users"`
}

func (u *UsersConfig) ExtractUsersMap() (UsersMap, error) {
	if u == nil {
		return nil, fmt.Errorf("users config is nil")
	}

	if len(u.Users) == 0 {
		return nil, fmt.Errorf("users list is empty")
	}

	res := make(UsersMap, len(u.Users))
	var errs []string

	uniqUsers := make(map[string]int)
	uniqTokens := make(map[Token]int)

	for indx, user := range u.Users {
		if alreadyUserIndex, ok := uniqUsers[user.Name]; ok {
			errs = append(
				errs,
				fmt.Sprintf(
					"user %d with name %s already present on index %d",
					indx,
					user.Name,
					alreadyUserIndex,
				),
			)
		} else {
			uniqUsers[user.Name] = indx
		}

		if err := user.Validate(); err != nil {
			errs = append(errs, fmt.Sprintf("incorrect user %d: %s", indx, err.Error()))
			continue
		}

		if alreadyTokenIndex, ok := uniqTokens[user.Token]; ok {
			errs = append(
				errs,
				fmt.Sprintf(
					"token for user (%d with name %s) already present for user with index %d",
					indx,
					user.Name,
					alreadyTokenIndex,
				),
			)
		} else {
			uniqTokens[user.Token] = indx
		}

		res[user.Token] = user
	}

	if len(errs) > 0 {
		return nil, errors.Join("Cannot extract users map", errs)
	}

	return res, nil
}

type Config struct {
	UnixSocketPath string
	BindAddress    string
	Users          UsersMap
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

	if len(c.Users) == 0 {
		errs = append(errs, "users list is empty")
	}

	for token, user := range c.Users {
		if err := user.Validate(); err != nil {
			errs = append(errs, fmt.Sprintf("incorrect user '%s': %v", user.Name, err))
		}

		if err := token.Validate(); err != nil {
			errs = append(errs, fmt.Sprintf("incorrect token for user '%s': %v", user.Name, err))
		}

		if !token.Eq(string(user.Token)) {
			errs = append(errs, fmt.Sprintf("token in user map not equal for user token '%s'", user.Name))
		}
	}

	if len(errs) > 0 {
		return errors.Join("Server config invalid", errs)
	}

	return nil
}
