package repository

import (
	"context"

	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/objective"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/period"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/user"
)

// ObjectiveRepository defines persistence operations for the Objective aggregate.
type ObjectiveRepository interface {
	// FindByID returns the objective for the given ID, or (nil, nil) if not found.
	FindByID(ctx context.Context, id objective.ObjectiveId) (*objective.Objective, error)

	// FindByPeriodID returns all objectives for the given period, ordered by display_order.
	FindByPeriodID(ctx context.Context, periodId period.PeriodId) ([]objective.Objective, error)

	// FindByOwnerID returns all objectives owned by the given user.
	FindByOwnerID(ctx context.Context, ownerId user.UserId) ([]objective.Objective, error)

	// Save persists the objective (upsert).
	Save(ctx context.Context, o objective.Objective) error

	// Remove deletes the objective by ID. Cascades to key_results.
	Remove(ctx context.Context, id objective.ObjectiveId) error
}
