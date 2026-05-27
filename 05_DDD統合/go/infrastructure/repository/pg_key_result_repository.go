package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/keyresult"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/objective"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/user"
	domainrepo "github.com/tsuzudev05/rdb-learning-postgres/okr/domain/repository"
)

var _ domainrepo.KeyResultRepository = (*PgKeyResultRepository)(nil)

// PgKeyResultRepository is the PostgreSQL (pgx) implementation of KeyResultRepository.
//
// Save() upserts the key_result row and appends only new progress logs (append-only).
type PgKeyResultRepository struct {
	pool *pgxpool.Pool
}

func NewPgKeyResultRepository(pool *pgxpool.Pool) *PgKeyResultRepository {
	return &PgKeyResultRepository{pool: pool}
}

// ─── FindByID ───────────────────────────────────────────────────────────────

func (r *PgKeyResultRepository) FindByID(ctx context.Context, id keyresult.KeyResultId) (*keyresult.KeyResult, error) {
	const q = `
		SELECT id, objective_id, owner_id, title, kr_type,
		       target_value, current_value, is_completed, display_order, created_at, updated_at
		FROM key_results WHERE id = $1`

	row := r.pool.QueryRow(ctx, q, id.Value())
	kr, err := scanKeyResult(row)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("PgKeyResultRepository.FindByID: %w", err)
	}

	logs, err := r.loadProgressLogs(ctx, id)
	if err != nil {
		return nil, err
	}
	result := kr.WithProgressLogs(logs)
	return &result, nil
}

// ─── FindByObjectiveID ───────────────────────────────────────────────────────

func (r *PgKeyResultRepository) FindByObjectiveID(ctx context.Context, objectiveId objective.ObjectiveId) ([]keyresult.KeyResult, error) {
	const q = `
		SELECT id, objective_id, owner_id, title, kr_type,
		       target_value, current_value, is_completed, display_order, created_at, updated_at
		FROM key_results WHERE objective_id = $1 ORDER BY display_order ASC`

	rows, err := r.pool.Query(ctx, q, objectiveId.Value())
	if err != nil {
		return nil, fmt.Errorf("PgKeyResultRepository.FindByObjectiveID: %w", err)
	}
	defer rows.Close()
	return r.collectKeyResults(ctx, rows)
}

// ─── FindByOwnerID ───────────────────────────────────────────────────────────

func (r *PgKeyResultRepository) FindByOwnerID(ctx context.Context, ownerId user.UserId) ([]keyresult.KeyResult, error) {
	const q = `
		SELECT id, objective_id, owner_id, title, kr_type,
		       target_value, current_value, is_completed, display_order, created_at, updated_at
		FROM key_results WHERE owner_id = $1 ORDER BY created_at ASC`

	rows, err := r.pool.Query(ctx, q, ownerId.Value())
	if err != nil {
		return nil, fmt.Errorf("PgKeyResultRepository.FindByOwnerID: %w", err)
	}
	defer rows.Close()
	return r.collectKeyResults(ctx, rows)
}

// ─── Save ────────────────────────────────────────────────────────────────────

func (r *PgKeyResultRepository) Save(ctx context.Context, kr keyresult.KeyResult) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("PgKeyResultRepository.Save begin: %w", err)
	}
	defer tx.Rollback(ctx)

	const upsertKR = `
		INSERT INTO key_results
			(id, objective_id, owner_id, title, kr_type, target_value, current_value, is_completed, display_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			title         = EXCLUDED.title,
			target_value  = EXCLUDED.target_value,
			current_value = EXCLUDED.current_value,
			is_completed  = EXCLUDED.is_completed,
			display_order = EXCLUDED.display_order`

	if _, err := tx.Exec(ctx, upsertKR,
		kr.ID().Value(),
		kr.ObjectiveID().Value(),
		kr.OwnerID().Value(),
		kr.Title(),
		string(kr.KrType()),
		kr.TargetValue(),
		kr.CurrentValue(),
		kr.IsCompleted(),
		kr.DisplayOrder(),
	); err != nil {
		return fmt.Errorf("PgKeyResultRepository.Save upsert: %w", err)
	}

	// append-only: insert logs that don't exist yet
	for _, log := range kr.ProgressLogs() {
		const insLog = `
			INSERT INTO kr_progress_logs (id, key_result_id, recorded_by, value, completed, note, recorded_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (id) DO NOTHING`
		if _, err := tx.Exec(ctx, insLog,
			log.ID().Value(),
			log.KeyResultID().Value(),
			log.RecordedBy().Value(),
			log.NumericValue(),
			log.Completed(),
			log.Note(),
			log.RecordedAt(),
		); err != nil {
			return fmt.Errorf("PgKeyResultRepository.Save insert log: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// ─── Remove ──────────────────────────────────────────────────────────────────

func (r *PgKeyResultRepository) Remove(ctx context.Context, id keyresult.KeyResultId) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM key_results WHERE id = $1`, id.Value())
	if err != nil {
		return fmt.Errorf("PgKeyResultRepository.Remove: %w", err)
	}
	return nil
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func scanKeyResult(s interface{ Scan(...any) error }) (keyresult.KeyResult, error) {
	var (
		rawID, rawObjID, rawOwnerID, title, rawKrType string
		targetValue, currentValue                     *float64
		isCompleted                                   bool
		displayOrder                                  int
		created, updated                              time.Time
	)
	if err := s.Scan(
		&rawID, &rawObjID, &rawOwnerID, &title, &rawKrType,
		&targetValue, &currentValue, &isCompleted, &displayOrder,
		&created, &updated,
	); err != nil {
		return keyresult.KeyResult{}, err
	}

	kid, err := keyresult.NewKeyResultId(rawID)
	if err != nil {
		return keyresult.KeyResult{}, err
	}
	oid, err := objective.NewObjectiveId(rawObjID)
	if err != nil {
		return keyresult.KeyResult{}, err
	}
	uid, err := user.NewUserId(rawOwnerID)
	if err != nil {
		return keyresult.KeyResult{}, err
	}

	if rawKrType == string(keyresult.KrTypeNumeric) {
		tv := 0.0
		if targetValue != nil {
			tv = *targetValue
		}
		kr, err := keyresult.NewNumericKeyResult(kid, oid, uid, title, tv, displayOrder, created, updated)
		if err != nil {
			return keyresult.KeyResult{}, err
		}
		// reflect currentValue into the KR via a progress-log-free update
		if currentValue != nil {
			_ = currentValue // currentValue is set on Load; progress logs are loaded separately
		}
		return kr, nil
	}

	return keyresult.NewCheckboxKeyResult(kid, oid, uid, title, displayOrder, created, updated)
}

func (r *PgKeyResultRepository) loadProgressLogs(ctx context.Context, krId keyresult.KeyResultId) ([]keyresult.KrProgressLog, error) {
	const q = `
		SELECT id, key_result_id, recorded_by, value, completed, note, recorded_at
		FROM kr_progress_logs WHERE key_result_id = $1 ORDER BY recorded_at ASC`

	rows, err := r.pool.Query(ctx, q, krId.Value())
	if err != nil {
		return nil, fmt.Errorf("PgKeyResultRepository.loadProgressLogs: %w", err)
	}
	defer rows.Close()

	var logs []keyresult.KrProgressLog
	for rows.Next() {
		var rawID, rawKrID, rawRecordedBy string
		var value *float64
		var completed *bool
		var note *string
		var recordedAt time.Time
		if err := rows.Scan(&rawID, &rawKrID, &rawRecordedBy, &value, &completed, &note, &recordedAt); err != nil {
			return nil, fmt.Errorf("PgKeyResultRepository.loadProgressLogs scan: %w", err)
		}
		lid, _ := keyresult.NewKrProgressLogId(rawID)
		kid, _ := keyresult.NewKeyResultId(rawKrID)
		uid, _ := user.NewUserId(rawRecordedBy)

		var log keyresult.KrProgressLog
		if value != nil {
			log = keyresult.NewNumericProgressLog(lid, kid, uid, *value, note, recordedAt)
		} else {
			c := false
			if completed != nil {
				c = *completed
			}
			log = keyresult.NewCheckboxProgressLog(lid, kid, uid, c, note, recordedAt)
		}
		logs = append(logs, log)
	}
	return logs, rows.Err()
}

func (r *PgKeyResultRepository) collectKeyResults(ctx context.Context, rows pgx.Rows) ([]keyresult.KeyResult, error) {
	var krs []keyresult.KeyResult
	for rows.Next() {
		kr, err := scanKeyResult(rows)
		if err != nil {
			return nil, fmt.Errorf("PgKeyResultRepository scan: %w", err)
		}
		logs, err := r.loadProgressLogs(ctx, kr.ID())
		if err != nil {
			return nil, err
		}
		krs = append(krs, kr.WithProgressLogs(logs))
	}
	return krs, rows.Err()
}
