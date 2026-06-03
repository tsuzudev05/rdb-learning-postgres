package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/period"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/infrastructure/repository"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/internal/testhelper"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

func mustPeriodId(t *testing.T, v string) period.PeriodId {
	t.Helper()
	id, err := period.NewPeriodId(v)
	if err != nil {
		t.Fatalf("NewPeriodId(%q): %v", v, err)
	}
	return id
}

func buildPeriod(t *testing.T, rawID, teamRawID, name, halfStr, start, end string) period.Period {
	t.Helper()
	id := mustPeriodId(t, rawID)
	tid := mustTeamId(t, teamRawID)
	half, err := period.NewHalf(halfStr)
	if err != nil {
		t.Fatalf("NewHalf(%q): %v", halfStr, err)
	}
	dr, err := period.NewDateRange(start, end)
	if err != nil {
		t.Fatalf("NewDateRange: %v", err)
	}
	p, err := period.NewPeriod(id, tid, name, half, dr, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("NewPeriod: %v", err)
	}
	return p
}

func mustSavePeriod(t *testing.T, ctx context.Context, repo interface {
	Save(context.Context, period.Period) error
}, p period.Period) {
	t.Helper()
	if err := repo.Save(ctx, p); err != nil {
		t.Fatalf("Save(%v): %v", p.ID().Value(), err)
	}
}

// seedTeam saves a team and returns it (no members).
func seedTeam(t *testing.T, ctx context.Context, rawID, name string) {
	t.Helper()
	tm := buildTeam(t, rawID, name)
	repo := repository.NewPgTeamRepository(testhelper.Pool)
	if err := repo.Save(ctx, tm); err != nil {
		t.Fatalf("seedTeam Save: %v", err)
	}
}

// ─── Save / FindByID ─────────────────────────────────────────────────────────

func TestPgPeriodRepository_Save_and_FindByID(t *testing.T) {
	t.Cleanup(func() { testhelper.TruncateAll(t, testhelper.Pool) })
	ctx := context.Background()
	seedTeam(t, ctx, "660e8400-e29b-41d4-a716-446655440001", "Alpha")

	repo := repository.NewPgPeriodRepository(testhelper.Pool)
	p := buildPeriod(t,
		"880e8400-e29b-41d4-a716-446655440001",
		"660e8400-e29b-41d4-a716-446655440001",
		"2026 上期", "H1", "2026-04-01", "2026-09-30",
	)
	mustSavePeriod(t, ctx, repo, p)

	got, err := repo.FindByID(ctx, p.ID())
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got == nil {
		t.Fatal("FindByID: got nil, want period")
	}
	if got.Name() != "2026 上期" {
		t.Errorf("Name = %q, want %q", got.Name(), "2026 上期")
	}
	if got.Half().Value() != "H1" {
		t.Errorf("Half = %q, want H1", got.Half().Value())
	}
}

func TestPgPeriodRepository_FindByID_NotFound(t *testing.T) {
	t.Cleanup(func() { testhelper.TruncateAll(t, testhelper.Pool) })
	ctx := context.Background()
	repo := repository.NewPgPeriodRepository(testhelper.Pool)

	id := mustPeriodId(t, "880e8400-e29b-41d4-a716-446655440099")
	got, err := repo.FindByID(ctx, id)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got != nil {
		t.Errorf("FindByID: got %v, want nil", got)
	}
}

// ─── FindByTeamID ────────────────────────────────────────────────────────────

func TestPgPeriodRepository_FindByTeamID(t *testing.T) {
	t.Cleanup(func() { testhelper.TruncateAll(t, testhelper.Pool) })
	ctx := context.Background()
	seedTeam(t, ctx, "660e8400-e29b-41d4-a716-446655440001", "Alpha")
	seedTeam(t, ctx, "660e8400-e29b-41d4-a716-446655440002", "Beta")

	repo := repository.NewPgPeriodRepository(testhelper.Pool)

	// Alpha に2期間追加
	mustSavePeriod(t, ctx, repo, buildPeriod(t,
		"880e8400-e29b-41d4-a716-446655440001",
		"660e8400-e29b-41d4-a716-446655440001",
		"2026 上期", "H1", "2026-04-01", "2026-09-30",
	))
	mustSavePeriod(t, ctx, repo, buildPeriod(t,
		"880e8400-e29b-41d4-a716-446655440002",
		"660e8400-e29b-41d4-a716-446655440001",
		"2026 下期", "H2", "2026-10-01", "2027-03-31",
	))
	// Beta に1期間追加
	mustSavePeriod(t, ctx, repo, buildPeriod(t,
		"880e8400-e29b-41d4-a716-446655440003",
		"660e8400-e29b-41d4-a716-446655440002",
		"2026 上期", "H1", "2026-04-01", "2026-09-30",
	))

	tid := mustTeamId(t, "660e8400-e29b-41d4-a716-446655440001")
	periods, err := repo.FindByTeamID(ctx, tid)
	if err != nil {
		t.Fatalf("FindByTeamID: %v", err)
	}
	if len(periods) != 2 {
		t.Errorf("FindByTeamID len = %d, want 2", len(periods))
	}
}

// ─── FindAll ─────────────────────────────────────────────────────────────────

func TestPgPeriodRepository_FindAll_Empty(t *testing.T) {
	t.Cleanup(func() { testhelper.TruncateAll(t, testhelper.Pool) })
	ctx := context.Background()
	repo := repository.NewPgPeriodRepository(testhelper.Pool)

	all, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("FindAll len = %d, want 0", len(all))
	}
}

func TestPgPeriodRepository_FindAll(t *testing.T) {
	t.Cleanup(func() { testhelper.TruncateAll(t, testhelper.Pool) })
	ctx := context.Background()
	seedTeam(t, ctx, "660e8400-e29b-41d4-a716-446655440001", "Alpha")

	repo := repository.NewPgPeriodRepository(testhelper.Pool)
	mustSavePeriod(t, ctx, repo, buildPeriod(t,
		"880e8400-e29b-41d4-a716-446655440001",
		"660e8400-e29b-41d4-a716-446655440001",
		"2026 上期", "H1", "2026-04-01", "2026-09-30",
	))
	mustSavePeriod(t, ctx, repo, buildPeriod(t,
		"880e8400-e29b-41d4-a716-446655440002",
		"660e8400-e29b-41d4-a716-446655440001",
		"2026 下期", "H2", "2026-10-01", "2027-03-31",
	))

	all, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("FindAll len = %d, want 2", len(all))
	}
}

// ─── Remove ──────────────────────────────────────────────────────────────────

func TestPgPeriodRepository_Remove(t *testing.T) {
	t.Cleanup(func() { testhelper.TruncateAll(t, testhelper.Pool) })
	ctx := context.Background()
	seedTeam(t, ctx, "660e8400-e29b-41d4-a716-446655440001", "Alpha")

	repo := repository.NewPgPeriodRepository(testhelper.Pool)
	p := buildPeriod(t,
		"880e8400-e29b-41d4-a716-446655440001",
		"660e8400-e29b-41d4-a716-446655440001",
		"2026 上期", "H1", "2026-04-01", "2026-09-30",
	)
	mustSavePeriod(t, ctx, repo, p)

	if err := repo.Remove(ctx, p.ID()); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	got, err := repo.FindByID(ctx, p.ID())
	if err != nil {
		t.Fatalf("FindByID after Remove: %v", err)
	}
	if got != nil {
		t.Error("Remove: period still exists after Remove")
	}
}

func TestPgPeriodRepository_Remove_Idempotent(t *testing.T) {
	t.Cleanup(func() { testhelper.TruncateAll(t, testhelper.Pool) })
	ctx := context.Background()
	repo := repository.NewPgPeriodRepository(testhelper.Pool)

	id := mustPeriodId(t, "880e8400-e29b-41d4-a716-446655440099")
	if err := repo.Remove(ctx, id); err != nil {
		t.Errorf("Remove (non-existent): want no error, got %v", err)
	}
}
