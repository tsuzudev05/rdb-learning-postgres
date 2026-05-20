package user

import "time"

// User is the aggregate root for the User aggregate.
// Immutable after construction — use factory function only.
type User struct {
	id           UserId
	name         string
	email        Email
	passwordHash string
	createdAt    time.Time
	updatedAt    time.Time
}

// NewUser constructs a User entity.
// createdAt / updatedAt are optional (zero value = not yet persisted).
func NewUser(
	id UserId,
	name string,
	email Email,
	passwordHash string,
	createdAt time.Time,
	updatedAt time.Time,
) (User, error) {
	if name == "" {
		return User{}, errorf("User: 氏名は空にできません")
	}
	return User{
		id:           id,
		name:         name,
		email:        email,
		passwordHash: passwordHash,
		createdAt:    createdAt,
		updatedAt:    updatedAt,
	}, nil
}

// ID returns the user's unique identifier.
func (u User) ID() UserId { return u.id }

// Name returns the user's display name.
func (u User) Name() string { return u.name }

// Email returns the user's email value object.
func (u User) Email() Email { return u.email }

// PasswordHash returns the stored password hash.
func (u User) PasswordHash() string { return u.passwordHash }

// CreatedAt returns the creation timestamp (zero if not persisted).
func (u User) CreatedAt() time.Time { return u.createdAt }

// UpdatedAt returns the last update timestamp (zero if not persisted).
func (u User) UpdatedAt() time.Time { return u.updatedAt }
