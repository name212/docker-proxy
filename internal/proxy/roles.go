package proxy

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
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
	Order       int
	Description string
	AllowPaths  []*AllowPath
}

func newRole(order int, desc string, paths []*AllowPath, inherit ...*Role) *Role {
	var allPaths []*AllowPath

	for _, i := range inherit {
		allPaths = append(allPaths, i.AllowPaths...)
	}

	allPaths = append(allPaths, paths...)

	return &Role{
		Order:       order,
		Description: desc,
		AllowPaths:  allPaths,
	}
}

func createMethodsOneRegexp(methods []string) []*regexp.Regexp {
	trimmed := make([]string, 0, len(methods))
	for _, m := range methods {
		trimmed = append(trimmed, strings.TrimSpace(m))
	}

	joined := strings.Join(trimmed, "|")
	reStr := fmt.Sprintf(`(?i)^(%s)$`, joined)

	return []*regexp.Regexp{
		regexp.MustCompile(reStr),
	}
}

var allowedRoles map[string]*Role

func init() {
	healthChecker := newRole(
		999,
		"access to health method as ping",
		[]*AllowPath{
			newAllowPath(
				regexp.MustCompile("/_ping"),
				createMethodsOneRegexp([]string{
					http.MethodGet,
					http.MethodHead,
				}),
			),
		},
	)

	allowedRoles = map[string]*Role{
		"healthChecker": healthChecker,
		"root": newRole(
			0,
			"full access to docker API",
			[]*AllowPath{
				newAllowPath(
					regexp.MustCompile(".+"),
					[]*regexp.Regexp{
						regexp.MustCompile(".+"),
					},
				),
			},
		),
	}

}

func RolesDescriptions() []string {
	type roleDescElem struct {
		name  string
		order int
		desc  string
	}

	list := make([]roleDescElem, 0, len(allowedRoles))

	for name, r := range allowedRoles {
		list = append(list, roleDescElem{
			name:  name,
			order: r.Order,
			desc:  r.Description,
		})
	}

	slices.SortFunc(list, func(i, j roleDescElem) int {
		return i.order - j.order
	})

	res := make([]string, 0, len(list))
	for _, d := range list {
		res = append(res, fmt.Sprintf(
			"%s - %s", d.name, d.desc,
		))
	}

	return res
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
