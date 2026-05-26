package repository

import (
	"context"

	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/team"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/user"
)

// TeamRepository defines persistence operations for the Team aggregate.
type TeamRepository interface {
	// FindByID returns the team (with members) for the given ID, or (nil, nil) if not found.
	FindByID(ctx context.Context, id team.TeamId) (*team.Team, error)

	// FindByUserID returns all teams the given user belongs to.
	FindByUserID(ctx context.Context, uid user.UserId) ([]team.Team, error)

	// FindAll returns all teams ordered by creation date (ascending).
	FindAll(ctx context.Context) ([]team.Team, error)

	// Save persists the team and its members (upsert + member full-replace).
	Save(ctx context.Context, t team.Team) error

	// Remove deletes the team by ID. Cascades to team_members.
	Remove(ctx context.Context, id team.TeamId) error
}
