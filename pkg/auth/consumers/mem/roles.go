package mem

import (
	"context"
	"fmt"
	"iter"
	"maps"
	"slices"

	"github.com/name212/docker-proxy/pkg/auth/roles"
)

type InMemoryRolesConsumer struct {
	defaultRoles roles.RolesMap

	roles roles.RolesMap
}

func NewInMemoryRolesConsumer(defaultRoles roles.RolesMap) (*InMemoryRolesConsumer, error) {
	if len(defaultRoles) == 0 {
		return nil, fmt.Errorf("MewInMemoryRolesConsumer: empty default roles")
	}

	if err := roles.ValidateRolesMap(defaultRoles); err != nil {
		return nil, fmt.Errorf("InMemoryRolesConsumer: default roles map: %w", err)
	}

	res := &InMemoryRolesConsumer{
		defaultRoles: make(roles.RolesMap, len(defaultRoles)),
		roles:        make(roles.RolesMap, len(defaultRoles)),
	}

	for name, r := range defaultRoles {
		cpy := r.Clone()

		res.defaultRoles[name] = cpy
		res.roles[name] = cpy
	}

	return res, nil
}

func (c *InMemoryRolesConsumer) GetOrders(ctx context.Context, rolesNames []string) (map[string]int, error) {
	res := make(map[string]int, len(rolesNames))
	var notFound []string

	for _, roleName := range rolesNames {
		r, ok := c.roles[roleName]
		if !ok {
			notFound = append(notFound, roleName)
			continue
		}

		res[roleName] = r.Order
	}

	if len(notFound) > 0 {
		return nil, fmt.Errorf("%w: %v", roles.ErrRoleNotFound, notFound)
	}

	return res, nil
}

func (c *InMemoryRolesConsumer) AllRolesNames(ctx context.Context) (iter.Seq[string], error) {
	return maps.Keys(c.roles), nil
}

func (c *InMemoryRolesConsumer) GetRoles(ctx context.Context, names []string) (roles.RolesMap, error) {
	var notFound []string

	res := make(roles.RolesMap, len(names))

	for _, n := range names {
		r, ok := c.roles[n]
		if !ok {
			notFound = append(notFound, n)
			continue
		}

		if len(notFound) == 0 {
			res[n] = r.Clone()
		}
	}

	if len(notFound) > 0 {
		return nil, fmt.Errorf("%w: %v", roles.ErrRoleNotFound, notFound)
	}

	return res, nil
}

func (c *InMemoryRolesConsumer) AddAdditionalRoles(additionalRoles roles.RolesMap) error {
	if err := roles.ValidateRolesMap(additionalRoles); err != nil {
		return fmt.Errorf("additional roles map is incorrect: %w", err)
	}

	var alreadyPresent []string

	for name := range additionalRoles {
		_, ok := c.roles[name]
		if ok {
			alreadyPresent = append(alreadyPresent, name)
		}
	}

	if len(alreadyPresent) > 0 {
		return fmt.Errorf("next roles already present. cannot override: %v", alreadyPresent)
	}

	for name, r := range additionalRoles {
		c.roles[name] = r.Clone()
	}

	return nil
}

func (c *InMemoryRolesConsumer) GetDefaultRoles(ctx context.Context) (roles.RolesMap, error) {
	if len(c.defaultRoles) == 0 {
		return nil, fmt.Errorf("default roles is empty")
	}

	res := make(roles.RolesMap, len(c.defaultRoles))

	for name, r := range c.defaultRoles {
		res[name] = r.Clone()
	}

	return res, nil
}

func (c *InMemoryRolesConsumer) RolesDescriptions() []string {
	type roleDescElem struct {
		name  string
		order int
		desc  string
	}

	list := make([]roleDescElem, 0, len(c.roles))

	for name, r := range c.roles {
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
