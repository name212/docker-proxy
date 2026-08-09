package proxy

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/name212/docker-proxy/internal/utils/errors"
	"github.com/name212/docker-proxy/internal/utils/re"
	ustrings "github.com/name212/docker-proxy/internal/utils/strings"
)

var (
	UnauthorizedErr = fmt.Errorf("unauthorized")
	versionPrefixRe = regexp.MustCompile(`^/v\d\.\d{1,3}`)
)

type AllowPath struct {
	Re             *re.Regexp   `yaml:"pathRegexp"`
	AllowMethodsRe []*re.Regexp `yaml:"allowMethodsRegexps"`
}

func (p *AllowPath) Clone() *AllowPath { 
	methods := make([]*re.Regexp, 0, len(p.AllowMethodsRe))
	for _, m := range p.AllowMethodsRe {
		methods = append(methods, m.Clone())
	}

	return &AllowPath{
		Re: p.Re.Clone(),
		AllowMethodsRe: methods,
	}
}

func (p *AllowPath) Validate() error {
	var errs []string

	if p.Re == nil {
		errs = append(errs, "pathRegexp is not passed")
	} else {
		if p.Re.String() == "" {
			errs = append(errs, "pathRegexp is empty")
		}
	}

	if len(p.AllowMethodsRe) == 0 {
		errs = append(errs, "allowMethodsRegexps not passed")
	} else {
		for i, m := range p.AllowMethodsRe {
			if m == nil {
				errs = append(errs, fmt.Sprintf("allowMethodsRegexps[%d] not passed", i))
				continue
			}

			if m.String() == "" {
				errs = append(errs, fmt.Sprintf("allowMethodsRegexps[%d] is empty", i))
				continue
			}
		}
	}

	if len(errs) > 0 {
		return errors.Join("AllowPath has next errors", errs)
	}

	return nil
}

func newAllowPath(re *re.Regexp, methods []*re.Regexp) *AllowPath {
	return &AllowPath{
		Re:             re,
		AllowMethodsRe: methods,
	}
}

type Role struct {
	Order       int          `yaml:"order"`
	Description string       `yaml:"description"`
	AllowPaths  []*AllowPath `yaml:"allowPaths"`
}

func (r *Role) Validate() error {
	if len(r.AllowPaths) == 0 {
		return fmt.Errorf("allowPaths is empty")
	}

	var errs []string

	for i, p := range r.AllowPaths {
		if p == nil {
			errs = append(errs, fmt.Sprintf("allowPaths[%d] not passed", i))
			continue
		}

		if err := p.Validate(); err != nil {
			errs = append(errs, fmt.Sprintf("allowPaths[%d] is invalid:\n%s", i, err.Error()))
			continue
		}
	}

	if len(errs) > 0 {
		return errors.Join("role is invalid", errs)
	}

	return nil
}

func (r *Role) Clone() *Role {
	allows := make([]*AllowPath, 0, len(r.AllowPaths))

	for _, p := range r.AllowPaths {
		allows = append(allows, p.Clone())
	}

	return  &Role{
		Order: r.Order,
		Description: r.Description,
		AllowPaths: allows,
	}
}

func NewRole(order int, desc string, paths []*AllowPath, inherit ...*Role) *Role {
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

func createMethodsOneRegexp(methods []string) []*re.Regexp {
	trimmed := make([]string, 0, len(methods))
	for _, m := range methods {
		trimmed = append(trimmed, strings.TrimSpace(m))
	}

	joined := strings.Join(trimmed, "|")
	reStr := fmt.Sprintf(`(?i)^(%s)$`, joined)

	return []*re.Regexp{
		re.MustCompile(reStr),
	}
}

type RolesMap map[string]*Role

var defaultAllowedRolesList ustrings.Set
var allowedRoles RolesMap

func init() {
	healthChecker := NewRole(
		999,
		"access to health method as ping",
		[]*AllowPath{
			newAllowPath(
				re.MustCompile("/_ping"),
				createMethodsOneRegexp([]string{
					http.MethodGet,
					http.MethodHead,
				}),
			),
		},
	)

	allowedRoles = RolesMap{
		"healthChecker": healthChecker,
		"root": NewRole(
			0,
			"full access to docker API",
			[]*AllowPath{
				newAllowPath(
					re.MustCompile(".+"),
					[]*re.Regexp{
						re.MustCompile(".+"),
					},
				),
			},
		),
	}

	defaultAllowedRolesList = ustrings.NewSetFromMap(allowedRoles)
}

func GetDefaultRolesList() ustrings.Set {
	return ustrings.NewSet(defaultAllowedRolesList)
}

func GetDefaultRoles() RolesMap {
	res := make(RolesMap, len(defaultAllowedRolesList))

	for name := range defaultAllowedRolesList {
		r, ok := allowedRoles[name]
		if !ok {
			panic(fmt.Sprintf("default role '%s' not found in roles map", name))
		}

		res[name] = r.Clone()
	}

	return res
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
