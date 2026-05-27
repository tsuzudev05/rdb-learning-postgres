package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/period"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/team"
	domainrepo "github.com/tsuzudev05/rdb-learning-postgres/okr/domain/repository"
)

var _ domainrepo.PeriodRepository = (*PgPeriodRepository)(nil)

// PgPeriodRepository is the PostgreSQL (pgx) implementation of PeriodRepository.
type PgPeriodRepository struct {
	pool *pgxpool.Pool
}

func NewPgPeriodRepository(pool *pgxpool.Pool) *PgPeriodRepository {
	return &PgPeriodRepository{pool: pool}
}

// ─── FindByID ───────────────────────────────────────────────────────────────

func (r *PgPeriodRepository) FindByID(ctx context.Context, id period.PeriodId) (*period.Period, error) {
	const q = `
		SELECT id, team_id, name, half, start_date, end_date, created_at, updated_at
		FROM periods WHERE id = $1`

	row := r.pool.QueryRow(ctx, q, id.Value())
	p, err := scanPeriod(row)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("PgPeriodRepository.FindByID: %w", err)
	}
	return &p, nil
}

// ─── FindByTeamID ────────────────────────────────────────────────────────────

func (r *PgPeriodRepository) FindByTeamID(ctx context.Context, teamId team.TeamId) ([]period.Period, error) {
	const q = `
		SELECT id, team_id, name, half, start_date, end_date, created_at, updated_at
		FROM periods WHERE team_id = $1 ORDER BY start_date ASC`

	rows, err := r.pool.Query(ctx, q, teamId.Value())
	if err != nil {
		return nil, fmt.Errorf("PgPeriodRepository.FindByTeamID: %w", err)
	}
	defer rows.Close()
	return collectPeriods(rows)
}

// ─── FindAll ─────────────────────────────────────────────────────────────────

func (r *PgPeriodRepository) FindAll(ctx context.Context) ([]period.Period, error) {
	const q = `
		SELECT id, team_id, name, half, start_date, end_date, created_at, updated_at
		FROM periods ORDER BY created_at ASC`

	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("PgPeriodRepository.FindAll: %w", err)
	}
	defer rows.Close()
	return collectPeriods(rows)
}

// ─── Save (upsert) ──────────────────────────────────────────────────────────

func (r *PgPeriodRepository) Save(ctx context.Context, p period.Period) error {
	const q = `
		INSERT INTO periods (id, team_id, name, half, start_date, end_date)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			name       = EXCLUDED.name,
			half       = EXCLUDED.half,
			start_date = EXCLUDED.start_date,
			end_date   = EXCLUDED.end_date`

	_, err := r.pool.Exec(ctx, q,
		p.ID().Value(),
		p.TeamID().Value(),
		p.Name(),
		p.Half().Value(),
		p.DateRange().Start(),
		p.DateRange().End(),
	)
	if err != nil {
		return fmt.Errorf("PgPeriodRepository.Save: %w", err)
	}
	return nil
}

// ─── Remove ──────────────────────────────────────────────────────────────────

func (r *PgPeriodRepository) Remove(ctx context.Context, id period.PeriodId) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM periods WHERE id = $1`, id.Value())
	if err != nil {
		return fmt.Errorf("PgPeriodRepository.Remove: %w", err)
	}
	return nil
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func scanPeriod(s interface{ Scan(...any) error }) (period.Period, error) {
	var (
		rawID, rawTeamID, name, rawHalf string
		startDate, endDate              time.Time
		created, updated                time.Time
	)
	if err := s.Scan(&rawID, &rawTeamID, &name, &rawHalf, &startDate, &endDate, &created, &updated); err != nil {
		return period.Period{}, err
	}
	pid, err := period.NewPeriodId(rawID)
	if err != nil {
		return period.Period{}, err
	}
	tid, err := team.NewTeamId(rawTeamID)
	if err != nil {
		return period.Period{}, err
	}
	h, err := period.NewHalf(rawHalf)
	if err != nil {
		return period.Period{}, err
	}
	dr, err := period.NewDateRangeFromTime(startDate, endDate)
	if err != nil {
		return period.Period{}, err
	}
	return period.NewPeriod(pid, tid, name, h, dr, created, updated)
}

func collectPeriods(rows pgx.Rows) ([]period.Period, error) {
	var periods []period.Period
	for rows.Next() {
		p, err := scanPeriod(rows)
		if err != nil {
			return nil, fmt.Errorf("PgPeriodRepository scan: %w", err)
		}
		periods = append(periods, p)
	}
	return periods, rows.Err()
}
