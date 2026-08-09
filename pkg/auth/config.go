package auth

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/name212/docker-proxy/pkg/auth/roles"
	"github.com/name212/docker-proxy/pkg/auth/users"
	"github.com/name212/docker-proxy/pkg/utils/errors"
	"github.com/name212/docker-proxy/pkg/utils/strings"
	ustrings "github.com/name212/docker-proxy/pkg/utils/strings"
)

var (
	DefaultRolesTimeout = 10 * time.Second
)

type DefaultRolesConsumer interface {
	GetDefaultRoles(ctx context.Context) (roles.RolesMap, error)
}

type CustomRole struct {
	*roles.Role `yaml:",inline"`

	InheritRoles []string `yaml:"inheritRoles"`
}

type UsersConfig struct {
	CustomRoles map[string]CustomRole `yaml:"customRoles"`
	Users       []*users.User         `yaml:"users"`
}

func (u *UsersConfig) ExtractCustomRoles(ctx context.Context, consumer DefaultRolesConsumer) (roles.RolesMap, error) {
	if len(u.CustomRoles) == 0 {
		return make(roles.RolesMap), nil
	}

	getDefaultRoles := func() (roles.RolesMap, error) {
		rolesCtx, cancel := context.WithTimeout(ctx, DefaultRolesTimeout)
		defer cancel()

		return consumer.GetDefaultRoles(rolesCtx)
	}

	allRolesMap, err := getDefaultRoles()
	if err != nil {
		return nil, fmt.Errorf("cannot get default roles: %w", err)
	}

	defRoles := strings.NewSetFromMap(allRolesMap)
	allRolesList := ustrings.NewSet(defRoles)

	var errs []string

	type roleToPrepare struct {
		name     string
		role     *roles.Role
		inherits ustrings.Set
	}

	rolesList := make([]*roleToPrepare, 0, len(u.CustomRoles))

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
			return -1
		}

		if i.inherits.Has(j.name) {
			return 1
		}

		if i.role != nil && j.role != nil {
			return i.role.Order - j.role.Order
		}

		return 0
	})

	errs = nil

	res := make(roles.RolesMap, len(rolesList))

	for _, r := range rolesList {
		roleToSet := r.role
		if roleToSet == nil {
			roleToSet = roles.NewRole(99999, "", nil)
		}
		inherits := make([]*roles.Role, 0, len(r.inherits))
		for inh := range r.inherits {
			inhRole, ok := allRolesMap[inh]
			if !ok {
				errs = append(errs, fmt.Sprintf("inhered role '%s' not found for role '%s'", inh, r.name))
				continue
			}

			inherits = append(inherits, inhRole.Clone())
		}

		resultRole := roles.NewRole(
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

func (u *UsersConfig) ExtractUsersMap(ctx context.Context, customer users.RolesConsumer) (users.UsersMap, error) {
	if u == nil {
		return nil, fmt.Errorf("users config is nil")
	}

	if len(u.Users) == 0 {
		return nil, fmt.Errorf("users list is empty")
	}

	res := make(users.UsersMap, len(u.Users))
	var errs []string

	uniqUsers := make(map[string]int)
	uniqTokens := make(map[users.Token]int)

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

		if err := user.Validate(ctx, customer); err != nil {
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
