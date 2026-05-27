package period

import (
	"fmt"
	"time"
)

// DateRange is a value object representing a start-end date pair.
// Invariant: start < end.
type DateRange struct {
	start time.Time
	end   time.Time
}

// NewDateRange creates a DateRange from "YYYY-MM-DD" strings.
func NewDateRange(start, end string) (DateRange, error) {
	const layout = "2006-01-02"
	s, err := time.Parse(layout, start)
	if err != nil {
		return DateRange{}, fmt.Errorf("DateRange: 開始日の形式が不正です: %s", start)
	}
	e, err := time.Parse(layout, end)
	if err != nil {
		return DateRange{}, fmt.Errorf("DateRange: 終了日の形式が不正です: %s", end)
	}
	if !s.Before(e) {
		return DateRange{}, fmt.Errorf("DateRange: 開始日は終了日より前でなければなりません")
	}
	return DateRange{start: s, end: e}, nil
}

// NewDateRangeFromTime creates a DateRange directly from time.Time values.
func NewDateRangeFromTime(start, end time.Time) (DateRange, error) {
	if !start.Before(end) {
		return DateRange{}, fmt.Errorf("DateRange: 開始日は終了日より前でなければなりません")
	}
	return DateRange{start: start, end: end}, nil
}

func (d DateRange) Start() time.Time { return d.start }
func (d DateRange) End() time.Time   { return d.end }

func (d DateRange) StartString() string {
	return d.start.Format("2006-01-02")
}
func (d DateRange) EndString() string {
	return d.end.Format("2006-01-02")
}
