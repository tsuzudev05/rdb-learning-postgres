package keyresult

import "fmt"

// KrProgressLogId is a value object for a KrProgressLog's unique identifier (UUID v4).
type KrProgressLogId struct {
	value string
}

func NewKrProgressLogId(value string) (KrProgressLogId, error) {
	if !uuidV4Regex.MatchString(value) {
		return KrProgressLogId{}, fmt.Errorf("KrProgressLogId: 無効なUUID形式です: %s", value)
	}
	return KrProgressLogId{value: value}, nil
}

func (k KrProgressLogId) Value() string                  { return k.value }
func (k KrProgressLogId) Equal(o KrProgressLogId) bool   { return k.value == o.value }
func (k KrProgressLogId) String() string                 { return k.value }
