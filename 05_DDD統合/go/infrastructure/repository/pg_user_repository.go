// Package repository provides PostgreSQL implementations of domain repository interfaces.
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/user"
	domainrepo "github.com/tsuzudev05/rdb-learning-postgres/okr/domain/repository"
)

// Compile-time check: PgUserRepository must satisfy UserRepository.
var _ domainrepo.UserRepository = (*PgUserRepository)(nil)

// PgUserRepository is the PostgreSQL (pgx) implementation of UserRepository.
//
// SQL policy:
//   - Save():   INSERT ... ON CONFLICT (id) DO UPDATE SET ... (upsert)
//   - Remove(): DELETE; no-op when the row does not exist
//   - Timestamps (created_at / updated_at) are managed by DB DEFAULT / trigger
type PgUserRepository struct {
	pool *pgxpool.Pool
}

// NewPgUserRepository creates a new PgUserRepository backed by the given connection pool.
func NewPgUserRepository(pool *pgxpool.Pool) *PgUserRepository {
	return &PgUserRepository{pool: pool}
}

// ─── FindByID ───────────────────────────────────────────────────────────────

func (r *PgUserRepository) FindByID(ctx context.Context, id user.UserId) (*user.User, error) {
	const q = `
		SELECT id, name, email, password_hash, created_at, updated_at
		FROM users
		WHERE id = $1`

	row := r.pool.QueryRow(ctx, q, id.Value())
	u, err := scanUser(row)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("PgUserRepository.FindByID: %w", err)
	}
	return u, nil
}

// ─── FindByEmail ────────────────────────────────────────────────────────────

func (r *PgUserRepository) FindByEmail(ctx context.Context, email user.Email) (*user.User, error) {
	const q = `
		SELECT id, name, email, password_hash, created_at, updated_at
		FROM users
		WHERE email = $1`

	row := r.pool.QueryRow(ctx, q, email.Value())
	u, err := scanUser(row)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("PgUserRepository.FindByEmail: %w", err)
	}
	return u, nil
}

// ─── FindAll ────────────────────────────────────────────────────────────────

func (r *PgUserRepository) FindAll(ctx context.Context) ([]user.User, error) {
	const q = `
		SELECT id, name, email, password_hash, created_at, updated_at
		FROM users
		ORDER BY created_at ASC`

	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("PgUserRepository.FindAll: %w", err)
	}
	defer rows.Close()

	var users []user.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("PgUserRepository.FindAll (scan): %w", err)
		}
		users = append(users, *u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("PgUserRepository.FindAll (rows): %w", err)
	}
	return users, nil
}

// ─── Save (upsert) ──────────────────────────────────────────────────────────

func (r *PgUserRepository) Save(ctx context.Context, u user.User) error {
	const q = `
		INSERT INTO users (id, name, email, password_hash)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE SET
			name          = EXCLUDED.name,
			email         = EXCLUDED.email,
			password_hash = EXCLUDED.password_hash`

	_, err := r.pool.Exec(ctx, q,
		u.ID().Value(),
		u.Name(),
		u.Email().Value(),
		u.PasswordHash(),
	)
	if err != nil {
		return fmt.Errorf("PgUserRepository.Save: %w", err)
	}
	return nil
}

// ─── Remove ─────────────────────────────────────────────────────────────────

func (r *PgUserRepository) Remove(ctx context.Context, id user.UserId) error {
	const q = `DELETE FROM users WHERE id = $1`
	_, err := r.pool.Exec(ctx, q, id.Value())
	if err != nil {
		return fmt.Errorf("PgUserRepository.Remove: %w", err)
	}
	return nil
}

// ─── helpers ────────────────────────────────────────────────────────────────

// scanner is satisfied by both pgx.Row and pgx.Rows.
type scanner interface {
	Scan(dest ...any) error
}

// scanUser reads one row into a User entity.
func scanUser(s scanner) (*user.User, error) {
	var (
		rawID    string
		name     string
		rawEmail string
		hash     string
		created  time.Time
		updated  time.Time
	)
	if err := s.Scan(&rawID, &name, &rawEmail, &hash, &created, &updated); err != nil {
		return nil, err
	}

	id, err := user.NewUserId(rawID)
	if err != nil {
		return nil, err
	}
	email, err := user.NewEmail(rawEmail)
	if err != nil {
		return nil, err
	}
	u, err := user.NewUser(id, name, email, hash, created, updated)
	if err != nil {
		return nil, err
	}
	return &u, nil
}
