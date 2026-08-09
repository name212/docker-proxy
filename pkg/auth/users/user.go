package users

import (
	"context"
	"fmt"
	"iter"
	"slices"
	"time"

	"github.com/name212/docker-proxy/pkg/utils/errors"
)

const defaultOrder int = 999999

var (
	RolesConsumerTimeout = 10 * time.Second

	ErrTokenNotMatch = fmt.Errorf("token not match")
	ErrUserNotFound  = fmt.Errorf("user not found by token")
)

type RolesConsumer interface {
	GetOrders(ctx context.Context, roles []string) (map[string]int, error)
	AllRolesNames(ctx context.Context) (iter.Seq[string], error)
}

type Token string
type UsersMap = map[Token]*User

func (t Token) Validate() error {
	if t == "" {
		return fmt.Errorf("token is empty")
	}

	if len(t) < 16 {
		return fmt.Errorf("token is short should minimum 16 chars")
	}

	return nil
}

func (t Token) Eq(passed string) bool {
	return t == Token(passed)
}

type User struct {
	Name  string   `yaml:"name"`
	Token Token    `yaml:"token"`
	Roles []string `yaml:"roles"`
}

func (u *User) Clone() *User {
	roles := make([]string, len(u.Roles))
	copy(roles, u.Roles)

	return &User{
		Name:  u.Name,
		Token: u.Token,
		Roles: roles,
	}
}

func (u *User) Validate(ctx context.Context, consumer RolesConsumer) error {
	if u == nil {
		return fmt.Errorf("user is nil")
	}

	if u.Name == "" {
		return fmt.Errorf("user name is empty")
	}

	if err := u.Token.Validate(); err != nil {
		return fmt.Errorf("user '%s' token is not valid: %w", u.Name, err)
	}

	if len(u.Roles) == 0 {
		return fmt.Errorf("for user '%s' not set any roles", u.Name)
	}

	orders, err := func(rolesNames []string) (map[string]int, error) {
		orderCtx, cancel := context.WithTimeout(ctx, RolesConsumerTimeout)
		defer cancel()

		return consumer.GetOrders(orderCtx, rolesNames)
	}(u.Roles)

	if err != nil {
		return fmt.Errorf("cannot get roles orders for user '%s': %w", u.Name, err)
	}

	var incorrectRoles []string
	for _, r := range u.Roles {
		if _, ok := orders[r]; !ok {
			incorrectRoles = append(incorrectRoles, r)
			continue
		}
	}

	if len(incorrectRoles) > 0 {
		var availableRoles []string
		allRolesCtx, cancel := context.WithTimeout(ctx, RolesConsumerTimeout)
		defer cancel()

		availableRolesIter, err := consumer.AllRolesNames(allRolesCtx)
		if err != nil {
			availableRoles = []string{fmt.Sprintf("available roles not consumed: %s", err.Error())}
		} else {
			availableRoles = slices.Collect(availableRolesIter)
		}

		return fmt.Errorf(
			"for user '%s' passed next incorrect roles:\n%v\nAvailable roles:\n%v",
			u.Name,
			incorrectRoles,
			availableRoles,
		)
	}

	slices.SortFunc(u.Roles, func(i, j string) int {
		iOrder, ok := orders[i]
		if !ok {
			iOrder = defaultOrder
		}
		jOrder, ok := orders[i]
		if !ok {
			jOrder = defaultOrder
		}

		return iOrder - jOrder
	})

	return nil
}

func (u *User) RolesAsStr() string {
	return fmt.Sprintf("%v", u.Roles)
}

func ValidateUsersMap(ctx context.Context, usersMap UsersMap, consumer RolesConsumer) error {
	if len(usersMap) == 0 {
		return fmt.Errorf("empty users map")
	}

	var errs []string

	for token, user := range usersMap {
		if err := user.Validate(ctx, consumer); err != nil {
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
		return errors.Join("incorrect users map", errs)
	}

	return nil
}
