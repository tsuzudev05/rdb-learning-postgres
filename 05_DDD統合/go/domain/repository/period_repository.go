package repository

import (
	"context"

	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/period"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/team"
)

// PeriodRepository defines persistence operations for the Period aggregate.
type PeriodRepository interface {
	// FindByID returns the period for the given ID, or (nil, nil) if not found.
	FindByID(ctx context.Context, id period.PeriodId) (*period.Period, error)

	// FindByTeamID returns all periods for the given team.
	FindByTeamID(ctx context.Context, teamId team.TeamId) ([]period.Period, error)

	// FindAll returns all periods ordered by creation date (ascending).
	FindAll(ctx context.Context) ([]period.Period, error)

	// Save persists the period (upsert).
	Save(ctx context.Context, p period.Period) error

	// Remove deletes the period by ID. Cascades to objectives.
	Remove(ctx context.Context, id period.PeriodId) error
}
