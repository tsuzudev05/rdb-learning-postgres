package team

import "fmt"

// Role is a value object representing a team member's role.
// Valid values: "admin" | "member"
type Role struct {
	value string
}

func NewRole(value string) (Role, error) {
	if value != "admin" && value != "member" {
		return Role{}, fmt.Errorf("Role: 無効なロールです: %s (admin または member)", value)
	}
	return Role{value: value}, nil
}

func Admin() Role  { return Role{value: "admin"} }
func Member() Role { return Role{value: "member"} }

func (r Role) Value() string       { return r.value }
func (r Role) IsAdmin() bool       { return r.value == "admin" }
func (r Role) Equal(o Role) bool   { return r.value == o.value }
func (r Role) String() string      { return r.value }
