// Package repository defines Repository interfaces for the domain layer.
// No infrastructure dependencies are allowed here.
package repository

import (
	"context"

	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/user"
)

// UserRepository defines persistence operations for the User aggregate.
// Implementations live in the infrastructure layer.
type UserRepository interface {
	// FindByID returns the user with the given ID, or (nil, nil) if not found.
	FindByID(ctx context.Context, id user.UserId) (*user.User, error)

	// FindByEmail returns the user with the given email, or (nil, nil) if not found.
	FindByEmail(ctx context.Context, email user.Email) (*user.User, error)

	// FindAll returns all users ordered by creation date (ascending).
	FindAll(ctx context.Context) ([]user.User, error)

	// Save persists the user (INSERT ... ON CONFLICT DO UPDATE).
	Save(ctx context.Context, u user.User) error

	// Remove deletes the user by ID. No-op if the user does not exist.
	Remove(ctx context.Context, id user.UserId) error
}
