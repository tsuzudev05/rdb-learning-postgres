package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/team"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/user"
	domainrepo "github.com/tsuzudev05/rdb-learning-postgres/okr/domain/repository"
)

var _ domainrepo.TeamRepository = (*PgTeamRepository)(nil)

// PgTeamRepository is the PostgreSQL (pgx) implementation of TeamRepository.
//
// Save() uses upsert for teams and DELETE+INSERT for team_members (full replace).
type PgTeamRepository struct {
	pool *pgxpool.Pool
}

func NewPgTeamRepository(pool *pgxpool.Pool) *PgTeamRepository {
	return &PgTeamRepository{pool: pool}
}

// ─── FindByID ───────────────────────────────────────────────────────────────

func (r *PgTeamRepository) FindByID(ctx context.Context, id team.TeamId) (*team.Team, error) {
	const q = `
		SELECT id, name, description, created_at, updated_at
		FROM teams WHERE id = $1`

	row := r.pool.QueryRow(ctx, q, id.Value())
	t, err := scanTeam(row)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("PgTeamRepository.FindByID: %w", err)
	}

	members, err := r.loadMembers(ctx, id)
	if err != nil {
		return nil, err
	}
	result, err := team.NewTeam(t.ID(), t.Name(), t.Description(), members, t.CreatedAt(), t.UpdatedAt())
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ─── FindByUserID ────────────────────────────────────────────────────────────

func (r *PgTeamRepository) FindByUserID(ctx context.Context, uid user.UserId) ([]team.Team, error) {
	const q = `
		SELECT t.id, t.name, t.description, t.created_at, t.updated_at
		FROM teams t
		JOIN team_members tm ON tm.team_id = t.id
		WHERE tm.user_id = $1
		ORDER BY t.created_at ASC`

	rows, err := r.pool.Query(ctx, q, uid.Value())
	if err != nil {
		return nil, fmt.Errorf("PgTeamRepository.FindByUserID: %w", err)
	}
	defer rows.Close()
	return r.collectTeams(ctx, rows)
}

// ─── FindAll ─────────────────────────────────────────────────────────────────

func (r *PgTeamRepository) FindAll(ctx context.Context) ([]team.Team, error) {
	const q = `SELECT id, name, description, created_at, updated_at FROM teams ORDER BY created_at ASC`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("PgTeamRepository.FindAll: %w", err)
	}
	defer rows.Close()
	return r.collectTeams(ctx, rows)
}

// ─── Save (upsert team + full-replace members) ───────────────────────────────

func (r *PgTeamRepository) Save(ctx context.Context, t team.Team) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("PgTeamRepository.Save begin: %w", err)
	}
	defer tx.Rollback(ctx)

	// upsert team
	const upsertTeam = `
		INSERT INTO teams (id, name, description)
		VALUES ($1, $2, $3)
		ON CONFLICT (id) DO UPDATE SET
			name        = EXCLUDED.name,
			description = EXCLUDED.description`
	if _, err := tx.Exec(ctx, upsertTeam, t.ID().Value(), t.Name(), t.Description()); err != nil {
		return fmt.Errorf("PgTeamRepository.Save upsert team: %w", err)
	}

	// full-replace members
	if _, err := tx.Exec(ctx, `DELETE FROM team_members WHERE team_id = $1`, t.ID().Value()); err != nil {
		return fmt.Errorf("PgTeamRepository.Save delete members: %w", err)
	}
	for _, m := range t.Members() {
		const ins = `
			INSERT INTO team_members (id, team_id, user_id, role)
			VALUES ($1, $2, $3, $4)`
		if _, err := tx.Exec(ctx, ins,
			m.ID().Value(), m.TeamID().Value(), m.UserID().Value(), m.Role().Value(),
		); err != nil {
			return fmt.Errorf("PgTeamRepository.Save insert member: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// ─── Remove ──────────────────────────────────────────────────────────────────

func (r *PgTeamRepository) Remove(ctx context.Context, id team.TeamId) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM teams WHERE id = $1`, id.Value())
	if err != nil {
		return fmt.Errorf("PgTeamRepository.Remove: %w", err)
	}
	return nil
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func scanTeam(s interface{ Scan(...any) error }) (team.Team, error) {
	var (
		rawID   string
		name    string
		desc    *string
		created time.Time
		updated time.Time
	)
	if err := s.Scan(&rawID, &name, &desc, &created, &updated); err != nil {
		return team.Team{}, err
	}
	id, err := team.NewTeamId(rawID)
	if err != nil {
		return team.Team{}, err
	}
	return team.NewTeam(id, name, desc, nil, created, updated)
}

func (r *PgTeamRepository) loadMembers(ctx context.Context, teamId team.TeamId) ([]team.TeamMember, error) {
	const q = `
		SELECT id, team_id, user_id, role, joined_at
		FROM team_members WHERE team_id = $1 ORDER BY joined_at ASC`
	rows, err := r.pool.Query(ctx, q, teamId.Value())
	if err != nil {
		return nil, fmt.Errorf("PgTeamRepository.loadMembers: %w", err)
	}
	defer rows.Close()

	var members []team.TeamMember
	for rows.Next() {
		var rawID, rawTeamID, rawUserID, rawRole string
		var joinedAt time.Time
		if err := rows.Scan(&rawID, &rawTeamID, &rawUserID, &rawRole, &joinedAt); err != nil {
			return nil, fmt.Errorf("PgTeamRepository.loadMembers scan: %w", err)
		}
		mid, _ := team.NewTeamMemberId(rawID)
		tid, _ := team.NewTeamId(rawTeamID)
		uid, _ := user.NewUserId(rawUserID)
		role, _ := team.NewRole(rawRole)
		m, err := team.NewTeamMember(mid, tid, uid, role, joinedAt)
		if err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

func (r *PgTeamRepository) collectTeams(ctx context.Context, rows pgx.Rows) ([]team.Team, error) {
	var teams []team.Team
	for rows.Next() {
		t, err := scanTeam(rows)
		if err != nil {
			return nil, fmt.Errorf("PgTeamRepository scan: %w", err)
		}
		members, err := r.loadMembers(ctx, t.ID())
		if err != nil {
			return nil, err
		}
		full, err := team.NewTeam(t.ID(), t.Name(), t.Description(), members, t.CreatedAt(), t.UpdatedAt())
		if err != nil {
			return nil, err
		}
		teams = append(teams, full)
	}
	return teams, rows.Err()
}
