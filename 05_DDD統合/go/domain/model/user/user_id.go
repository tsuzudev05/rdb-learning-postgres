// Package user contains domain model types for the User aggregate.
package user

import (
	"fmt"
	"regexp"
)

// uuidV4Regex matches UUID v4 format.
var uuidV4Regex = regexp.MustCompile(
	`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`,
)

// UserId is a value object representing a user's unique identifier (UUID v4).
// Constructed only via Create() to enforce validation.
type UserId struct {
	value string
}

// NewUserId creates a UserId after validating the UUID v4 format.
func NewUserId(value string) (UserId, error) {
	if !uuidV4Regex.MatchString(value) {
		return UserId{}, fmt.Errorf("UserId: 無効なUUID形式です: %s", value)
	}
	return UserId{value: value}, nil
}

// Value returns the raw UUID string.
func (u UserId) Value() string { return u.value }

// Equal returns true if two UserIds have the same value.
func (u UserId) Equal(other UserId) bool { return u.value == other.value }

// String implements fmt.Stringer.
func (u UserId) String() string { return u.value }
