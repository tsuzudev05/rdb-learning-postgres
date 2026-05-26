package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/objective"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/period"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/user"
	domainrepo "github.com/tsuzudev05/rdb-learning-postgres/okr/domain/repository"
)

var _ domainrepo.ObjectiveRepository = (*PgObjectiveRepository)(nil)

// PgObjectiveRepository is the PostgreSQL (pgx) implementation of ObjectiveRepository.
type PgObjectiveRepository struct {
	pool *pgxpool.Pool
}

func NewPgObjectiveRepository(pool *pgxpool.Pool) *PgObjectiveRepository {
	return &PgObjectiveRepository{pool: pool}
}

// ─── FindByID ───────────────────────────────────────────────────────────────

func (r *PgObjectiveRepository) FindByID(ctx context.Context, id objective.ObjectiveId) (*objective.Objective, error) {
	const q = `
		SELECT id, period_id, owner_id, title, description, display_order, created_at, updated_at
		FROM objectives WHERE id = $1`

	row := r.pool.QueryRow(ctx, q, id.Value())
	o, err := scanObjective(row)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("PgObjectiveRepository.FindByID: %w", err)
	}
	return &o, nil
}

// ─── FindByPeriodID ──────────────────────────────────────────────────────────

func (r *PgObjectiveRepository) FindByPeriodID(ctx context.Context, periodId period.PeriodId) ([]objective.Objective, error) {
	const q = `
		SELECT id, period_id, owner_id, title, description, display_order, created_at, updated_at
		FROM objectives WHERE period_id = $1 ORDER BY display_order ASC`

	rows, err := r.pool.Query(ctx, q, periodId.Value())
	if err != nil {
		return nil, fmt.Errorf("PgObjectiveRepository.FindByPeriodID: %w", err)
	}
	defer rows.Close()
	return collectObjectives(rows)
}

// ─── FindByOwnerID ───────────────────────────────────────────────────────────

func (r *PgObjectiveRepository) FindByOwnerID(ctx context.Context, ownerId user.UserId) ([]objective.Objective, error) {
	const q = `
		SELECT id, period_id, owner_id, title, description, display_order, created_at, updated_at
		FROM objectives WHERE owner_id = $1 ORDER BY created_at ASC`

	rows, err := r.pool.Query(ctx, q, ownerId.Value())
	if err != nil {
		return nil, fmt.Errorf("PgObjectiveRepository.FindByOwnerID: %w", err)
	}
	defer rows.Close()
	return collectObjectives(rows)
}

// ─── Save (upsert) ──────────────────────────────────────────────────────────

func (r *PgObjectiveRepository) Save(ctx context.Context, o objective.Objective) error {
	const q = `
		INSERT INTO objectives (id, period_id, owner_id, title, description, display_order)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			title         = EXCLUDED.title,
			description   = EXCLUDED.description,
			display_order = EXCLUDED.display_order`

	_, err := r.pool.Exec(ctx, q,
		o.ID().Value(),
		o.PeriodID().Value(),
		o.OwnerID().Value(),
		o.Title(),
		o.Description(),
		o.DisplayOrder(),
	)
	if err != nil {
		return fmt.Errorf("PgObjectiveRepository.Save: %w", err)
	}
	return nil
}

// ─── Remove ──────────────────────────────────────────────────────────────────

func (r *PgObjectiveRepository) Remove(ctx context.Context, id objective.ObjectiveId) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM objectives WHERE id = $1`, id.Value())
	if err != nil {
		return fmt.Errorf("PgObjectiveRepository.Remove: %w", err)
	}
	return nil
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func scanObjective(s interface{ Scan(...any) error }) (objective.Objective, error) {
	var (
		rawID, rawPeriodID, rawOwnerID, title string
		desc                                  *string
		displayOrder                          int
		created, updated                      time.Time
	)
	if err := s.Scan(&rawID, &rawPeriodID, &rawOwnerID, &title, &desc, &displayOrder, &created, &updated); err != nil {
		return objective.Objective{}, err
	}
	oid, err := objective.NewObjectiveId(rawID)
	if err != nil {
		return objective.Objective{}, err
	}
	pid, err := period.NewPeriodId(rawPeriodID)
	if err != nil {
		return objective.Objective{}, err
	}
	uid, err := user.NewUserId(rawOwnerID)
	if err != nil {
		return objective.Objective{}, err
	}
	return objective.NewObjective(oid, pid, uid, title, desc, displayOrder, created, updated)
}

func collectObjectives(rows pgx.Rows) ([]objective.Objective, error) {
	var objs []objective.Objective
	for rows.Next() {
		o, err := scanObjective(rows)
		if err != nil {
			return nil, fmt.Errorf("PgObjectiveRepository scan: %w", err)
		}
		objs = append(objs, o)
	}
	return objs, rows.Err()
}
