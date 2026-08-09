package mem

import (
	"context"
	"fmt"

	"github.com/name212/docker-proxy/pkg/auth/users"
	"github.com/name212/docker-proxy/pkg/utils/errors"
)

type InMemoryUsersConsumer struct {
	usersMap users.UsersMap
}

func NewInMemoryUsersConsumer(ctx context.Context, usersMap users.UsersMap, consumer users.RolesConsumer) (*InMemoryUsersConsumer, error) {
	if err := users.ValidateUsersMap(ctx, usersMap, consumer); err != nil {
		return nil, fmt.Errorf("InMemoryUsersConsumer: %w", err)
	}

	return &InMemoryUsersConsumer{
		usersMap: usersMap,
	}, nil
}

func (c *InMemoryUsersConsumer) GetUser(ctx context.Context, token users.Token) (*users.User, error) {
	u, ok := c.usersMap[token]
	if !ok {
		return nil, users.ErrUserNotFound
	}

	return u.Clone(), nil
}

func (c *InMemoryUsersConsumer) AddUser(ctx context.Context, usersMap users.UsersMap, consumer users.RolesConsumer) error {
	if err := users.ValidateUsersMap(ctx, usersMap, consumer); err != nil {
		return fmt.Errorf("InMemoryUsersConsumer: cannot add users: %w", err)
	}

	var alreadyExists []string

	for token, u := range usersMap {
		if foundUser, ok := c.usersMap[token]; ok {
			alreadyExists = append(alreadyExists, fmt.Sprintf("same tokens '%s':'%s'", u.Name, foundUser.Name))
		}

		for _, presentUser := range c.usersMap {
			if u.Name == presentUser.Name {
				alreadyExists = append(alreadyExists, fmt.Sprintf("user with name '%s' already exists", u.Name))
			}
		}
	}

	if len(alreadyExists) > 0 {
		return errors.Join("found duplicate users or tokens", alreadyExists)
	}

	for token, u := range usersMap {
		c.usersMap[token] = u
	}

	return nil
}
