// Package team contains domain model types for the Team aggregate.
package team

import (
	"fmt"
	"regexp"
)

var uuidV4Regex = regexp.MustCompile(
	`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`,
)

// TeamId is a value object representing a team's unique identifier (UUID v4).
type TeamId struct {
	value string
}

func NewTeamId(value string) (TeamId, error) {
	if !uuidV4Regex.MatchString(value) {
		return TeamId{}, fmt.Errorf("TeamId: 無効なUUID形式です: %s", value)
	}
	return TeamId{value: value}, nil
}

func (t TeamId) Value() string          { return t.value }
func (t TeamId) Equal(o TeamId) bool    { return t.value == o.value }
func (t TeamId) String() string         { return t.value }
