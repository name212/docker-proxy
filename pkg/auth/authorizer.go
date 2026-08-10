package auth

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"

	"github.com/name212/docker-proxy/pkg/auth/roles"
	"github.com/name212/docker-proxy/pkg/auth/users"
)

var (
	ErrGetUser       = fmt.Errorf("cannot get user")
	ErrGetRoles      = fmt.Errorf("cannot get roles")
	ErrIncorrectPath = fmt.Errorf("incorrect path")
	ErrUnauthorized  = fmt.Errorf("unauthorized")
)

type PathPreparator func(path string) string

type RolesConsumer interface {
	GetRoles(ctx context.Context, names []string) (roles.RolesMap, error)
}

type UserConsumer interface {
	GetUser(ctx context.Context, token users.Token) (*users.User, error)
}

func NoPathPreparator(p string) string {
	return p
}

func DropByRegexpPathPreparator(r *regexp.Regexp) PathPreparator {
	return func(path string) string {
		return r.ReplaceAllString(path, "")
	}
}

type AllowResult struct {
	User         string
	Role         string
	PathMatcher  string
	MethodMather string
}

func (r *AllowResult) String() string {
	return fmt.Sprintf(
		"user='%s' role='%s' pathMatcher='%s' methodMatcher='%s",
		r.User,
		r.Role,
		r.PathMatcher,
		r.MethodMather,
	)
}

type Authorizer struct {
	rolesConsumer  RolesConsumer
	userConsumer   UserConsumer
	pathPreparator PathPreparator
}

func NewAuthorizer(rolesConsumer RolesConsumer, userConsumer UserConsumer, pathPreparator PathPreparator) *Authorizer {
	if pathPreparator == nil {
		pathPreparator = NoPathPreparator
	}

	return &Authorizer{
		rolesConsumer:  rolesConsumer,
		userConsumer:   userConsumer,
		pathPreparator: pathPreparator,
	}
}

func (a *Authorizer) Allow(ctx context.Context, token users.Token, u *url.URL, method string) (*AllowResult, error) {
	if method == "" {
		return nil, fmt.Errorf("%w: empty method", ErrIncorrectPath)
	}

	path := a.pathPreparator(u.Path)
	if path == "" {
		return nil, fmt.Errorf("%w: empty path", ErrIncorrectPath)
	}

	user, err := a.userConsumer.GetUser(ctx, token)
	if err != nil {
		if errors.Is(err, users.ErrUserNotFound) {
			return nil, fmt.Errorf("%w: %w", err, ErrUnauthorized)
		}

		return nil, fmt.Errorf("%w: %w: %w", ErrGetUser, err, ErrUnauthorized)
	}

	if user == nil {
		return nil, fmt.Errorf("%w got nil user (not found): %w", ErrGetUser, ErrUnauthorized)
	}

	userName := user.Name

	if len(user.Roles) == 0 {
		return nil, errForUser(userName, "%w not assign any role for use", ErrGetUser)
	}

	cpyRolesNames := make([]string, len(user.Roles))
	copy(cpyRolesNames, user.Roles)

	allowedRoles, err := a.rolesConsumer.GetRoles(ctx, cpyRolesNames)
	if err != nil {
		return nil, errForUser(userName, "%w: %w", ErrGetRoles, err)
	}

	var incorrectRoles []string

	var allowRes *AllowResult

	for _, roleStr := range user.Roles {
		role, ok := allowedRoles[roleStr]
		if !ok {
			incorrectRoles = append(incorrectRoles, roleStr)
			continue
		}

		if len(role.AllowPaths) == 0 {
			incorrectRoles = append(
				incorrectRoles,
				"allows paths list empty for role '%s'",
				roleStr,
			)
			continue
		}

		for _, expectedPath := range role.AllowPaths {
			if len(expectedPath.AllowMethodsRe) == 0 {
				incorrectRoles = append(
					incorrectRoles,
					"allows methods list empty for role '%s' for path %s",
					roleStr,
					expectedPath.Re.String(),
				)
				continue
			}

			if !expectedPath.Re.MatchString(path) {
				continue
			}

			for _, expectedMethod := range expectedPath.AllowMethodsRe {
				if expectedMethod.MatchString(method) {
					allowRes = &AllowResult{
						User:         user.Name,
						Role:         roleStr,
						PathMatcher:  expectedPath.Re.String(),
						MethodMather: expectedMethod.String(),
					}
				}
			}
		}
	}

	if len(incorrectRoles) > 0 {
		return nil, errForUser(userName, "%w: not found or incorrect roles %v", ErrGetRoles, incorrectRoles)
	}

	if allowRes != nil {
		return allowRes, nil
	}

	return nil, errForUser(userName, "%w", ErrUnauthorized)
}

func errForUser(userName string, f string, args... any) error {
	uMsg := fmt.Sprintf("auth for user '%s': ", userName)

	return fmt.Errorf(uMsg + f, args...)
}