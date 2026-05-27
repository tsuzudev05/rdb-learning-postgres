package repository

import (
	"context"

	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/keyresult"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/objective"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/user"
)

// KeyResultRepository defines persistence operations for the KeyResult aggregate.
type KeyResultRepository interface {
	// FindByID returns the key result (with progress logs) for the given ID, or (nil, nil) if not found.
	FindByID(ctx context.Context, id keyresult.KeyResultId) (*keyresult.KeyResult, error)

	// FindByObjectiveID returns all key results for the given objective, ordered by display_order.
	FindByObjectiveID(ctx context.Context, objectiveId objective.ObjectiveId) ([]keyresult.KeyResult, error)

	// FindByOwnerID returns all key results owned by the given user.
	FindByOwnerID(ctx context.Context, ownerId user.UserId) ([]keyresult.KeyResult, error)

	// Save persists the key result and its progress logs (upsert KR + append-only logs).
	Save(ctx context.Context, kr keyresult.KeyResult) error

	// Remove deletes the key result by ID. Cascades to kr_progress_logs.
	Remove(ctx context.Context, id keyresult.KeyResultId) error
}
