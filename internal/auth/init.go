package auth

import (
	"context"
	"fmt"
	"regexp"
	"sync/atomic"

	"github.com/name212/docker-proxy/pkg/auth"
	authconsumers "github.com/name212/docker-proxy/pkg/auth/consumers/mem"
)

type RolesDescriptorI interface {
	RolesDescriptions() []string
}

var (
	versionPrefixRe = regexp.MustCompile(`^/v\d\.\d{1,3}`)

	rolesConsumer   *authconsumers.InMemoryRolesConsumer
	RolesDescriptor RolesDescriptorI

	alreadyInit atomic.Bool
)

func init() {
	var err error
	rolesConsumer, err = authconsumers.NewInMemoryRolesConsumer(getDefaultRoles())
	if err != nil {
		panic(fmt.Errorf("cannot initialize roles consumer with default roles"))
	}

	RolesDescriptor = rolesConsumer
}

func CreateAuthorizer(ctx context.Context, cfg *auth.UsersConfig) (*auth.Authorizer, error) {
	if !alreadyInit.CompareAndSwap(false, true) {
		return nil, fmt.Errorf("authorizer already created")
	}

	customRolesMap, err := cfg.ExtractCustomRoles(ctx, rolesConsumer)
	if err != nil {
		return nil, fmt.Errorf("cannot extract custom roles map from users config: %w", err)
	}

	if len(customRolesMap) > 0 {
		if err := rolesConsumer.AddAdditionalRoles(customRolesMap); err != nil {
			return nil, fmt.Errorf("cannot add custom roles: %w", err)
		}
	}

	usersMap, err := cfg.ExtractUsersMap(ctx, rolesConsumer)
	if err != nil {
		return nil, fmt.Errorf("cannot extract users map from users config: %w", err)
	}

	if len(usersMap) == 0 {
		return nil, fmt.Errorf("extracted users map is empty")
	}

	usersConsumer, err := authconsumers.NewInMemoryUsersConsumer(ctx, usersMap, rolesConsumer)
	if err != nil {
		return nil, fmt.Errorf("cannot create users consumer: %w", err)
	}

	removeVersionPathPreparator := auth.DropByRegexpPathPreparator(versionPrefixRe)

	return auth.NewAuthorizer(rolesConsumer, usersConsumer, removeVersionPathPreparator), nil
}
