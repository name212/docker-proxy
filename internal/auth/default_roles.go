// Copyright 2026
// license that can be found in the LICENSE file.

package auth

import (
	"net/http"

	"github.com/name212/docker-proxy/pkg/auth/roles"
	"github.com/name212/docker-proxy/pkg/utils/re"
)

func getDefaultRoles() roles.RolesMap {
	healthChecker := roles.NewRole(
		999,
		"access to health method as ping",
		[]*roles.AllowPath{
			roles.NewAllowPath(
				re.MustCompile("/_ping"),
				roles.CreateMethodsOneRegexp([]string{
					http.MethodGet,
					http.MethodHead,
				}),
			),
		},
	)

	return roles.RolesMap{
		"healthChecker": healthChecker,
		"root": roles.NewRole(
			0,
			"full access to docker API",
			[]*roles.AllowPath{
				roles.NewAllowPath(
					re.MustCompile(".+"),
					[]*re.Regexp{
						re.MustCompile(".+"),
					},
				),
			},
		),
	}
}
