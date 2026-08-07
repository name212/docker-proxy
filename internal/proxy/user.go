package proxy

import (
	"fmt"
	"maps"
	"slices"
)

type Token string

var TokenNotMatchErr = fmt.Errorf("token not match")

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

func (u *User) Validate() error {
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

	var incorrectRoles []string
	for _, r := range u.Roles {
		_, ok := allowedRoles[r]
		if !ok {
			incorrectRoles = append(incorrectRoles, r)
		}
	}

	if len(incorrectRoles) > 0 {
		availableRoles := slices.Collect(maps.Keys(allowedRoles))
		return fmt.Errorf(
			"for user '%s' passed next incorrect roles:\n%v\nAvailable roles:\n%v",
			u.Name,
			incorrectRoles,
			availableRoles,
		)
	}

	slices.SortFunc(u.Roles, func(i, j string) int {
		return allowedRoles[i].Order - allowedRoles[j].Order
	})

	return nil
}

func (u *User) RolesAsStr() string {
	return fmt.Sprintf("%v", u.Roles)
}

type UsersMap = map[Token]*User
