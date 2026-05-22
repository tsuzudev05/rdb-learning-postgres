package user

import (
	"fmt"
	"regexp"
	"strings"
)

// emailRegex performs a simplified RFC5322 check.
var emailRegex = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// Email is a value object representing a validated, lowercase email address.
type Email struct {
	value string
}

// NewEmail validates and normalises the email (lowercased).
func NewEmail(value string) (Email, error) {
	lower := strings.ToLower(strings.TrimSpace(value))
	if !emailRegex.MatchString(lower) {
		return Email{}, fmt.Errorf("Email: 無効なメールアドレスです: %s", value)
	}
	return Email{value: lower}, nil
}

// Value returns the normalised email string.
func (e Email) Value() string { return e.value }

// Equal returns true if two Emails have the same value.
func (e Email) Equal(other Email) bool { return e.value == other.value }

// String implements fmt.Stringer.
func (e Email) String() string { return e.value }
