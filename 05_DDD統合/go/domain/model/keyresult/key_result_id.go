// Package keyresult contains domain model types for the KeyResult aggregate.
package keyresult

import (
	"fmt"
	"regexp"
)

var uuidV4Regex = regexp.MustCompile(
	`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`,
)

// KeyResultId is a value object for a KeyResult's unique identifier (UUID v4).
type KeyResultId struct {
	value string
}

func NewKeyResultId(value string) (KeyResultId, error) {
	if !uuidV4Regex.MatchString(value) {
		return KeyResultId{}, fmt.Errorf("KeyResultId: 無効なUUID形式です: %s", value)
	}
	return KeyResultId{value: value}, nil
}

func (k KeyResultId) Value() string              { return k.value }
func (k KeyResultId) Equal(o KeyResultId) bool   { return k.value == o.value }
func (k KeyResultId) String() string             { return k.value }
