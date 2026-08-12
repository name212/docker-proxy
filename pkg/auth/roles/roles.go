// Copyright 2026
// license that can be found in the LICENSE file.

package roles

import (
	"fmt"
	"strings"

	"github.com/name212/docker-proxy/pkg/utils/errors"
	"github.com/name212/docker-proxy/pkg/utils/re"
)

var (
	ErrRoleNotFound = fmt.Errorf("role(s) not found")
)

type RolesMap map[string]*Role

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
		Re:             p.Re.Clone(),
		AllowMethodsRe: methods,
	}
}

func (p *AllowPath) Validate() error {
	var errs []string

	if p.Re == nil {
		errs = append(errs, "pathRegexp is not passed")
	} else if p.Re.String() == "" {
		errs = append(errs, "pathRegexp is empty")
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

func NewAllowPath(re *re.Regexp, methods []*re.Regexp) *AllowPath {
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

	return &Role{
		Order:       r.Order,
		Description: r.Description,
		AllowPaths:  allows,
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

func CreateMethodsOneRegexp(methods []string) []*re.Regexp {
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

func ValidateRolesMap(rolesMap RolesMap) error {
	var errs []string

	for name, r := range rolesMap {
		if err := r.Validate(); err != nil {
			errs = append(errs, fmt.Sprintf("role '%s' invalid: %s", name, err.Error()))
		}
	}

	if len(errs) > 0 {
		return errors.Join("roles map is invalid", errs)
	}

	return nil
}
