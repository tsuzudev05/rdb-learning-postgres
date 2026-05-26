// Package objective contains domain model types for the Objective aggregate.
package objective

import (
	"fmt"
	"regexp"
)

var uuidV4Regex = regexp.MustCompile(
	`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`,
)

// ObjectiveId is a value object representing an objective's unique identifier (UUID v4).
type ObjectiveId struct {
	value string
}

func NewObjectiveId(value string) (ObjectiveId, error) {
	if !uuidV4Regex.MatchString(value) {
		return ObjectiveId{}, fmt.Errorf("ObjectiveId: 無効なUUID形式です: %s", value)
	}
	return ObjectiveId{value: value}, nil
}

func (o ObjectiveId) Value() string             { return o.value }
func (o ObjectiveId) Equal(x ObjectiveId) bool  { return o.value == x.value }
func (o ObjectiveId) String() string            { return o.value }
