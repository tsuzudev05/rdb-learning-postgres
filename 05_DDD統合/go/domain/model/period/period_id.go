// Package period contains domain model types for the Period aggregate.
package period

import (
	"fmt"
	"regexp"
)

var uuidV4Regex = regexp.MustCompile(
	`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`,
)

// PeriodId is a value object representing a period's unique identifier (UUID v4).
type PeriodId struct {
	value string
}

func NewPeriodId(value string) (PeriodId, error) {
	if !uuidV4Regex.MatchString(value) {
		return PeriodId{}, fmt.Errorf("PeriodId: 無効なUUID形式です: %s", value)
	}
	return PeriodId{value: value}, nil
}

func (p PeriodId) Value() string          { return p.value }
func (p PeriodId) Equal(o PeriodId) bool  { return p.value == o.value }
func (p PeriodId) String() string         { return p.value }
