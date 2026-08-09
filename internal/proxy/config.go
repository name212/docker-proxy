package proxy

import (
	"fmt"
	"slices"

	"github.com/name212/docker-proxy/internal/utils/errors"
	ustrings "github.com/name212/docker-proxy/internal/utils/strings"
)

type CustomRole struct {
	*Role

	InheritRoles []string `yaml:"inheritRoles"`
}

type UsersConfig struct {
	CustomRoles map[string]CustomRole `yaml:"customRoles"`
	Users       []*User               `yaml:"users"`
}

func (u *UsersConfig) ExtractCustomRoles() (RolesMap, error) {
	if len(u.CustomRoles) == 0 {
		return make(RolesMap), nil
	}

	defRoles := GetDefaultRolesList()
	allRolesList := ustrings.NewSet(defRoles)
	allRolesMap := GetDefaultRoles()

	var errs []string

	type roleToPrepare struct {
		name     string
		role     *Role
		inherits ustrings.Set
	}

	rolesList := make([]*roleToPrepare, len(u.CustomRoles))

	inheritsOneByOne := make(map[string]ustrings.Set)

	for name, r := range u.CustomRoles {
		if defRoles.Has(name) {
			errs = append(errs, fmt.Sprintf("role '%s' already present as default. Cannot replace", name))
			continue
		}

		allRolesList.Add(name)

		if r.Role == nil && len(r.InheritRoles) == 0 {
			errs = append(errs, fmt.Sprintf("role '%s' cannot present role or inheritRoles", name))
			continue
		}

		if r.Role != nil {
			if err := r.Role.Validate(); err != nil {
				errs = append(errs, fmt.Sprintf("role '%s' cannot present role or inheritRoles", name))
			}
		}

		inherits := ustrings.NewSetFromSlice(r.InheritRoles)

		rolesList = append(rolesList, &roleToPrepare{
			name:     name,
			role:     r.Role,
			inherits: inherits,
		})

		inheritsOneByOne[name] = inherits
	}

	if len(errs) > 0 {
		return nil, errors.Join("incorrect custom roles", errs)
	}

	errs = nil

	// check cyclic deps and  all inherits presents

	depsToRole := ustrings.NewSet()

	for _, r := range rolesList {
		depsToRole.Clean()

		for anotherRole, anotherInherent := range inheritsOneByOne {
			if anotherRole == r.name {
				continue
			}

			if anotherInherent.Has(r.name) {
				depsToRole.Add(anotherRole)
			}
		}

		for iName := range r.inherits {
			if !allRolesList.Has(iName) {
				errs = append(
					errs,
					fmt.Sprintf(
						"inherit role '%s' for role '%s' not present in all roles",
						iName,
						r.name,
					),
				)
			}

			if depsToRole.Has(iName) {
				errs = append(
					errs,
					fmt.Sprintf(
						"role '%s' is cyclic for '%s'",
						r.name,
						iName,
					),
				)
			}
		}
	}

	if len(errs) > 0 {
		return nil, errors.Join("incorrect custom roles - cyclic deps or not found deps", errs)
	}

	slices.SortStableFunc(rolesList, func(i, j *roleToPrepare) int {
		if j.inherits.Has(i.name) {
			return 1
		}

		if i.inherits.Has(j.name) {
			return -1
		}

		if i.role != nil && j.role != nil {
			return i.role.Order - j.role.Order
		}

		return 0
	})

	errs = nil

	res := make(RolesMap, len(rolesList))

	for _, r := range rolesList {
		roleToSet := r.role
		if roleToSet == nil {
			roleToSet = NewRole(99999, "", nil)
		}
		inherits := make([]*Role, 0, len(r.inherits))
		for inh := range r.inherits {
			inhRole, ok := allRolesMap[inh]
			if !ok {
				errs = append(errs, fmt.Sprintf("inhered role '%s' not found for role '%s'", inh, r.name))
				continue
			}

			inherits = append(inherits, inhRole.Clone())
		}

		resultRole := NewRole(
			roleToSet.Order, 
			roleToSet.Description, 
			roleToSet.AllowPaths,
			inherits...,
		)

		if err := resultRole.Validate(); err != nil {
			errs = append(errs, fmt.Sprintf("result custom role '%s' invalid: %s", r.name, err.Error()))
			continue
		}

		res[r.name] = resultRole
		allRolesMap[r.name] = resultRole
	}

	if len(errs) > 0 {
		return nil, errors.Join("incorrect custom roles - cannot create result map", errs)
	}

	return res, nil
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
