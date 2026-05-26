package keyresult

import (
	"fmt"
	"time"

	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/objective"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/user"
)

// KrType represents the kind of progress tracking for a KeyResult.
type KrType string

const (
	KrTypeNumeric  KrType = "numeric"
	KrTypeCheckbox KrType = "checkbox"
)

// KeyResult is the aggregate root for the KeyResult (KR in OKR).
// Progress logs are managed inside the aggregate.
type KeyResult struct {
	id           KeyResultId
	objectiveId  objective.ObjectiveId
	ownerId      user.UserId
	title        string
	krType       KrType
	targetValue  *float64 // numeric のみ
	currentValue *float64 // numeric のみ
	isCompleted  bool     // checkbox のみ
	displayOrder int
	progressLogs []KrProgressLog
	createdAt    time.Time
	updatedAt    time.Time
}

func NewNumericKeyResult(
	id KeyResultId,
	objectiveId objective.ObjectiveId,
	ownerId user.UserId,
	title string,
	targetValue float64,
	displayOrder int,
	createdAt, updatedAt time.Time,
) (KeyResult, error) {
	if title == "" {
		return KeyResult{}, fmt.Errorf("KeyResult: タイトルは空にできません")
	}
	return KeyResult{
		id: id, objectiveId: objectiveId, ownerId: ownerId,
		title: title, krType: KrTypeNumeric,
		targetValue: &targetValue, currentValue: nil,
		displayOrder: displayOrder,
		progressLogs: []KrProgressLog{},
		createdAt: createdAt, updatedAt: updatedAt,
	}, nil
}

func NewCheckboxKeyResult(
	id KeyResultId,
	objectiveId objective.ObjectiveId,
	ownerId user.UserId,
	title string,
	displayOrder int,
	createdAt, updatedAt time.Time,
) (KeyResult, error) {
	if title == "" {
		return KeyResult{}, fmt.Errorf("KeyResult: タイトルは空にできません")
	}
	return KeyResult{
		id: id, objectiveId: objectiveId, ownerId: ownerId,
		title: title, krType: KrTypeCheckbox,
		isCompleted:  false,
		displayOrder: displayOrder,
		progressLogs: []KrProgressLog{},
		createdAt: createdAt, updatedAt: updatedAt,
	}, nil
}

func (k KeyResult) ID() KeyResultId                    { return k.id }
func (k KeyResult) ObjectiveID() objective.ObjectiveId { return k.objectiveId }
func (k KeyResult) OwnerID() user.UserId               { return k.ownerId }
func (k KeyResult) Title() string                      { return k.title }
func (k KeyResult) KrType() KrType                     { return k.krType }
func (k KeyResult) TargetValue() *float64              { return k.targetValue }
func (k KeyResult) CurrentValue() *float64             { return k.currentValue }
func (k KeyResult) IsCompleted() bool                  { return k.isCompleted }
func (k KeyResult) DisplayOrder() int                  { return k.displayOrder }
func (k KeyResult) ProgressLogs() []KrProgressLog      { return k.progressLogs }
func (k KeyResult) CreatedAt() time.Time               { return k.createdAt }
func (k KeyResult) UpdatedAt() time.Time               { return k.updatedAt }

// UpdateNumericProgress updates current_value and appends a progress log.
func (k *KeyResult) UpdateNumericProgress(log KrProgressLog) error {
	if k.krType != KrTypeNumeric {
		return fmt.Errorf("KeyResult.UpdateNumericProgress: numeric KR ではありません")
	}
	if log.NumericValue() == nil {
		return fmt.Errorf("KeyResult.UpdateNumericProgress: 数値が必要です")
	}
	k.currentValue = log.NumericValue()
	k.progressLogs = append(k.progressLogs, log)
	return nil
}

// UpdateCheckboxProgress updates is_completed and appends a progress log.
func (k *KeyResult) UpdateCheckboxProgress(log KrProgressLog) error {
	if k.krType != KrTypeCheckbox {
		return fmt.Errorf("KeyResult.UpdateCheckboxProgress: checkbox KR ではありません")
	}
	if log.Completed() == nil {
		return fmt.Errorf("KeyResult.UpdateCheckboxProgress: 完了フラグが必要です")
	}
	k.isCompleted = *log.Completed()
	k.progressLogs = append(k.progressLogs, log)
	return nil
}

// WithProgressLogs returns a new KeyResult with the given logs (used by Repository on load).
func (k KeyResult) WithProgressLogs(logs []KrProgressLog) KeyResult {
	k.progressLogs = logs
	return k
}
