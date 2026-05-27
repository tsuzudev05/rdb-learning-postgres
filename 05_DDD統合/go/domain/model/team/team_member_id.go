package team

import "fmt"

// TeamMemberId is a value object for a TeamMember's unique identifier (UUID v4).
type TeamMemberId struct {
	value string
}

func NewTeamMemberId(value string) (TeamMemberId, error) {
	if !uuidV4Regex.MatchString(value) {
		return TeamMemberId{}, fmt.Errorf("TeamMemberId: 無効なUUID形式です: %s", value)
	}
	return TeamMemberId{value: value}, nil
}

func (t TeamMemberId) Value() string             { return t.value }
func (t TeamMemberId) Equal(o TeamMemberId) bool { return t.value == o.value }
func (t TeamMemberId) String() string            { return t.value }
