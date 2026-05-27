package period

import "fmt"

// Half is a value object representing a half-year cycle.
// Valid values: "H1" | "H2"
type Half struct {
	value string
}

func NewHalf(value string) (Half, error) {
	if value != "H1" && value != "H2" {
		return Half{}, fmt.Errorf("Half: 無効な値です: %s (H1 または H2)", value)
	}
	return Half{value: value}, nil
}

func H1() Half { return Half{value: "H1"} }
func H2() Half { return Half{value: "H2"} }

func (h Half) Value() string      { return h.value }
func (h Half) Equal(o Half) bool  { return h.value == o.value }
func (h Half) String() string     { return h.value }
