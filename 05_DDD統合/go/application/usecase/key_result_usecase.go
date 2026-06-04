package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/keyresult"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/objective"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/user"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/repository"
)

// KeyResultUseCase handles application logic for the KeyResult aggregate.
type KeyResultUseCase struct {
	repo repository.KeyResultRepository
}

// NewKeyResultUseCase creates a KeyResultUseCase with the given repository.
func NewKeyResultUseCase(repo repository.KeyResultRepository) *KeyResultUseCase {
	return &KeyResultUseCase{repo: repo}
}

// CreateNumericKRInput holds input for creating a numeric KeyResult.
type CreateNumericKRInput struct {
	ObjectiveID  string
	OwnerID      string
	Title        string
	TargetValue  float64
	DisplayOrder int
}

// CreateNumericKeyResult creates and persists a numeric KeyResult.
func (uc *KeyResultUseCase) CreateNumericKeyResult(ctx context.Context, in CreateNumericKRInput) (*keyresult.KeyResult, error) {
	objID, err := objective.NewObjectiveId(in.ObjectiveID)
	if err != nil {
		return nil, err
	}
	ownerID, err := user.NewUserId(in.OwnerID)
	if err != nil {
		return nil, err
	}
	kid, err := keyresult.NewKeyResultId(uuid.New().String())
	if err != nil {
		return nil, fmt.Errorf("KeyResultUseCase.CreateNumericKeyResult: UUID生成エラー: %w", err)
	}

	now := time.Now().UTC()
	kr, err := keyresult.NewNumericKeyResult(kid, objID, ownerID, in.Title, in.TargetValue, in.DisplayOrder, now, now)
	if err != nil {
		return nil, err
	}

	if err := uc.repo.Save(ctx, kr); err != nil {
		return nil, fmt.Errorf("KeyResultUseCase.CreateNumericKeyResult: 保存エラー: %w", err)
	}
	return &kr, nil
}

// CreateCheckboxKRInput holds input for creating a checkbox KeyResult.
type CreateCheckboxKRInput struct {
	ObjectiveID  string
	OwnerID      string
	Title        string
	DisplayOrder int
}

// CreateCheckboxKeyResult creates and persists a checkbox KeyResult.
func (uc *KeyResultUseCase) CreateCheckboxKeyResult(ctx context.Context, in CreateCheckboxKRInput) (*keyresult.KeyResult, error) {
	objID, err := objective.NewObjectiveId(in.ObjectiveID)
	if err != nil {
		return nil, err
	}
	ownerID, err := user.NewUserId(in.OwnerID)
	if err != nil {
		return nil, err
	}
	kid, err := keyresult.NewKeyResultId(uuid.New().String())
	if err != nil {
		return nil, fmt.Errorf("KeyResultUseCase.CreateCheckboxKeyResult: UUID生成エラー: %w", err)
	}

	now := time.Now().UTC()
	kr, err := keyresult.NewCheckboxKeyResult(kid, objID, ownerID, in.Title, in.DisplayOrder, now, now)
	if err != nil {
		return nil, err
	}

	if err := uc.repo.Save(ctx, kr); err != nil {
		return nil, fmt.Errorf("KeyResultUseCase.CreateCheckboxKeyResult: 保存エラー: %w", err)
	}
	return &kr, nil
}

// UpdateNumericProgressInput holds input for updating numeric KR progress.
type UpdateNumericProgressInput struct {
	KeyResultID string
	RecordedBy  string
	Value       float64
	Note        *string
}

// UpdateNumericProgress fetches the KR, appends a progress log, and saves.
// Returns error if the KR is not found or is not a numeric KR.
func (uc *KeyResultUseCase) UpdateNumericProgress(ctx context.Context, in UpdateNumericProgressInput) (*keyresult.KeyResult, error) {
	kid, err := keyresult.NewKeyResultId(in.KeyResultID)
	if err != nil {
		return nil, err
	}
	recordedBy, err := user.NewUserId(in.RecordedBy)
	if err != nil {
		return nil, err
	}

	kr, err := uc.repo.FindByID(ctx, kid)
	if err != nil {
		return nil, fmt.Errorf("KeyResultUseCase.UpdateNumericProgress: KR取得エラー: %w", err)
	}
	if kr == nil {
		return nil, fmt.Errorf("KeyResultUseCase.UpdateNumericProgress: KRが見つかりません: %s", in.KeyResultID)
	}

	logID, err := keyresult.NewKrProgressLogId(uuid.New().String())
	if err != nil {
		return nil, fmt.Errorf("KeyResultUseCase.UpdateNumericProgress: UUID生成エラー: %w", err)
	}
	log := keyresult.NewNumericProgressLog(logID, kid, recordedBy, in.Value, in.Note, time.Now().UTC())

	if err := kr.UpdateNumericProgress(log); err != nil {
		return nil, fmt.Errorf("KeyResultUseCase.UpdateNumericProgress: %w", err)
	}

	if err := uc.repo.Save(ctx, *kr); err != nil {
		return nil, fmt.Errorf("KeyResultUseCase.UpdateNumericProgress: 保存エラー: %w", err)
	}
	return kr, nil
}

// UpdateCheckboxProgressInput holds input for updating checkbox KR progress.
type UpdateCheckboxProgressInput struct {
	KeyResultID string
	RecordedBy  string
	Completed   bool
	Note        *string
}

// UpdateCheckboxProgress fetches the KR, appends a progress log, and saves.
// Returns error if the KR is not found or is not a checkbox KR.
func (uc *KeyResultUseCase) UpdateCheckboxProgress(ctx context.Context, in UpdateCheckboxProgressInput) (*keyresult.KeyResult, error) {
	kid, err := keyresult.NewKeyResultId(in.KeyResultID)
	if err != nil {
		return nil, err
	}
	recordedBy, err := user.NewUserId(in.RecordedBy)
	if err != nil {
		return nil, err
	}

	kr, err := uc.repo.FindByID(ctx, kid)
	if err != nil {
		return nil, fmt.Errorf("KeyResultUseCase.UpdateCheckboxProgress: KR取得エラー: %w", err)
	}
	if kr == nil {
		return nil, fmt.Errorf("KeyResultUseCase.UpdateCheckboxProgress: KRが見つかりません: %s", in.KeyResultID)
	}

	logID, err := keyresult.NewKrProgressLogId(uuid.New().String())
	if err != nil {
		return nil, fmt.Errorf("KeyResultUseCase.UpdateCheckboxProgress: UUID生成エラー: %w", err)
	}
	log := keyresult.NewCheckboxProgressLog(logID, kid, recordedBy, in.Completed, in.Note, time.Now().UTC())

	if err := kr.UpdateCheckboxProgress(log); err != nil {
		return nil, fmt.Errorf("KeyResultUseCase.UpdateCheckboxProgress: %w", err)
	}

	if err := uc.repo.Save(ctx, *kr); err != nil {
		return nil, fmt.Errorf("KeyResultUseCase.UpdateCheckboxProgress: 保存エラー: %w", err)
	}
	return kr, nil
}
