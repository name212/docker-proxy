package proxy

import (
	"fmt"
	"net/url"
	"regexp"
)

var (
	UnauthorizedErr = fmt.Errorf("unauthorized")
	versionPrefixRe = regexp.MustCompile(`^/v\d\.\d{1,3}`)
)

type AllowPath struct {
	Re             *regexp.Regexp
	AllowMethodsRe []*regexp.Regexp
}

func newAllowPath(re *regexp.Regexp, methods []*regexp.Regexp) *AllowPath {
	return &AllowPath{
		Re:             re,
		AllowMethodsRe: methods,
	}
}

type Role struct {
	Order      int
	AllowPaths []*AllowPath
}

func newRole(order int, paths []*AllowPath) *Role {
	return &Role{
		Order:      order,
		AllowPaths: paths,
	}
}

var allowedRoles map[string]*Role = map[string]*Role{
	"admin": newRole(0, []*AllowPath{
		newAllowPath(
			regexp.MustCompile(".+"),
			[]*regexp.Regexp{
				regexp.MustCompile(".+"),
			},
		),
	}),
}

func IsAllow(user *User, u *url.URL, method string) (string, error) {
	if len(user.Roles) == 0 {
		return "", fmt.Errorf("not assign any role")
	}

	path := versionPrefixRe.ReplaceAllString(u.Path, "")

	if path == "" {
		return "", fmt.Errorf("empty path")
	}

	var incorrectRoles []string

	allow := false
	allowByPath := ""
	allowByMethod := ""
	allowByRole := ""

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
					allow = true
					allowByPath = expectedPath.Re.String()
					allowByMethod = expectedMethod.String()
					allowByRole = roleStr
					break
				}
			}
		}
	}

	if allow {
		return fmt.Sprintf(
			"role: '%s' path: '%s' method: '%s'",
			allowByRole,
			allowByPath,
			allowByMethod,
		), nil
	}

	return "", UnauthorizedErr
}
