package keyresult

import (
	"time"

	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/user"
)

// KrProgressLog is an entity within the KeyResult aggregate.
// It records one progress update (numeric value or checkbox).
type KrProgressLog struct {
	id           KrProgressLogId
	keyResultId  KeyResultId
	recordedBy   user.UserId
	numericValue *float64 // numeric KR のみ
	completed    *bool    // checkbox KR のみ
	note         *string
	recordedAt   time.Time
}

func NewNumericProgressLog(
	id KrProgressLogId,
	krId KeyResultId,
	recordedBy user.UserId,
	value float64,
	note *string,
	recordedAt time.Time,
) KrProgressLog {
	return KrProgressLog{
		id: id, keyResultId: krId, recordedBy: recordedBy,
		numericValue: &value, note: note, recordedAt: recordedAt,
	}
}

func NewCheckboxProgressLog(
	id KrProgressLogId,
	krId KeyResultId,
	recordedBy user.UserId,
	completed bool,
	note *string,
	recordedAt time.Time,
) KrProgressLog {
	return KrProgressLog{
		id: id, keyResultId: krId, recordedBy: recordedBy,
		completed: &completed, note: note, recordedAt: recordedAt,
	}
}

func (l KrProgressLog) ID() KrProgressLogId  { return l.id }
func (l KrProgressLog) KeyResultID() KeyResultId { return l.keyResultId }
func (l KrProgressLog) RecordedBy() user.UserId  { return l.recordedBy }
func (l KrProgressLog) NumericValue() *float64   { return l.numericValue }
func (l KrProgressLog) Completed() *bool         { return l.completed }
func (l KrProgressLog) Note() *string            { return l.note }
func (l KrProgressLog) RecordedAt() time.Time    { return l.recordedAt }
