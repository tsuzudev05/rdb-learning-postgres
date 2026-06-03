package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/objective"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/period"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/infrastructure/repository"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/internal/testhelper"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

func mustObjectiveId(t *testing.T, v string) objective.ObjectiveId {
	t.Helper()
	id, err := objective.NewObjectiveId(v)
	if err != nil {
		t.Fatalf("NewObjectiveId(%q): %v", v, err)
	}
	return id
}

func buildObjective(t *testing.T, rawID, periodRawID, ownerRawID, title string, order int) objective.Objective {
	t.Helper()
	id := mustObjectiveId(t, rawID)
	pid, err := period.NewPeriodId(periodRawID)
	if err != nil {
		t.Fatalf("NewPeriodId: %v", err)
	}
	oid := mustUserId(t, ownerRawID)
	desc := "テスト目標"
	o, err := objective.NewObjective(id, pid, oid, title, &desc, order, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("NewObjective: %v", err)
	}
	return o
}

func mustSaveObjective(t *testing.T, ctx context.Context, repo interface {
	Save(context.Context, objective.Objective) error
}, o objective.Objective) {
	t.Helper()
	if err := repo.Save(ctx, o); err != nil {
		t.Fatalf("Save(%v): %v", o.ID().Value(), err)
	}
}

// seedPeriod saves a period under a team and returns it.
func seedPeriod(t *testing.T, ctx context.Context, rawID, teamRawID, name, halfStr, start, end string) period.Period {
	t.Helper()
	p := buildPeriod(t, rawID, teamRawID, name, halfStr, start, end)
	repo := repository.NewPgPeriodRepository(testhelper.Pool)
	if err := repo.Save(ctx, p); err != nil {
		t.Fatalf("seedPeriod Save: %v", err)
	}
	return p
}

// ─── Save / FindByID ─────────────────────────────────────────────────────────

func TestPgObjectiveRepository_Save_and_FindByID(t *testing.T) {
	t.Cleanup(func() { testhelper.TruncateAll(t, testhelper.Pool) })
	ctx := context.Background()
	seedTeam(t, ctx, "660e8400-e29b-41d4-a716-446655440001", "Alpha")
	alice := seedUser(t, ctx, "550e8400-e29b-41d4-a716-446655440001", "Alice", "alice@example.com")
	seedPeriod(t, ctx, "880e8400-e29b-41d4-a716-446655440001",
		"660e8400-e29b-41d4-a716-446655440001", "2026 上期", "H1", "2026-04-01", "2026-09-30")

	repo := repository.NewPgObjectiveRepository(testhelper.Pool)
	o := buildObjective(t,
		"990e8400-e29b-41d4-a716-446655440001",
		"880e8400-e29b-41d4-a716-446655440001",
		alice.ID().Value(),
		"売上を前年比150%にする", 0,
	)
	mustSaveObjective(t, ctx, repo, o)

	got, err := repo.FindByID(ctx, o.ID())
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got == nil {
		t.Fatal("FindByID: got nil, want objective")
	}
	if got.Title() != "売上を前年比150%にする" {
		t.Errorf("Title = %q, want %q", got.Title(), "売上を前年比150%にする")
	}
}

func TestPgObjectiveRepository_FindByID_NotFound(t *testing.T) {
	t.Cleanup(func() { testhelper.TruncateAll(t, testhelper.Pool) })
	ctx := context.Background()
	repo := repository.NewPgObjectiveRepository(testhelper.Pool)

	id := mustObjectiveId(t, "990e8400-e29b-41d4-a716-446655440099")
	got, err := repo.FindByID(ctx, id)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got != nil {
		t.Errorf("FindByID: got %v, want nil", got)
	}
}

// ─── FindByPeriodID ──────────────────────────────────────────────────────────

func TestPgObjectiveRepository_FindByPeriodID(t *testing.T) {
	t.Cleanup(func() { testhelper.TruncateAll(t, testhelper.Pool) })
	ctx := context.Background()
	seedTeam(t, ctx, "660e8400-e29b-41d4-a716-446655440001", "Alpha")
	alice := seedUser(t, ctx, "550e8400-e29b-41d4-a716-446655440001", "Alice", "alice@example.com")
	p1 := seedPeriod(t, ctx, "880e8400-e29b-41d4-a716-446655440001",
		"660e8400-e29b-41d4-a716-446655440001", "2026 上期", "H1", "2026-04-01", "2026-09-30")
	p2 := seedPeriod(t, ctx, "880e8400-e29b-41d4-a716-446655440002",
		"660e8400-e29b-41d4-a716-446655440001", "2026 下期", "H2", "2026-10-01", "2027-03-31")

	repo := repository.NewPgObjectiveRepository(testhelper.Pool)

	// p1 に2件
	mustSaveObjective(t, ctx, repo, buildObjective(t,
		"990e8400-e29b-41d4-a716-446655440001", p1.ID().Value(), alice.ID().Value(), "OKR-1", 0))
	mustSaveObjective(t, ctx, repo, buildObjective(t,
		"990e8400-e29b-41d4-a716-446655440002", p1.ID().Value(), alice.ID().Value(), "OKR-2", 1))
	// p2 に1件
	mustSaveObjective(t, ctx, repo, buildObjective(t,
		"990e8400-e29b-41d4-a716-446655440003", p2.ID().Value(), alice.ID().Value(), "OKR-3", 0))

	objs, err := repo.FindByPeriodID(ctx, p1.ID())
	if err != nil {
		t.Fatalf("FindByPeriodID: %v", err)
	}
	if len(objs) != 2 {
		t.Errorf("FindByPeriodID len = %d, want 2", len(objs))
	}
}

// ─── FindByOwnerID ───────────────────────────────────────────────────────────

func TestPgObjectiveRepository_FindByOwnerID(t *testing.T) {
	t.Cleanup(func() { testhelper.TruncateAll(t, testhelper.Pool) })
	ctx := context.Background()
	seedTeam(t, ctx, "660e8400-e29b-41d4-a716-446655440001", "Alpha")
	alice := seedUser(t, ctx, "550e8400-e29b-41d4-a716-446655440001", "Alice", "alice@example.com")
	bob   := seedUser(t, ctx, "550e8400-e29b-41d4-a716-446655440002", "Bob", "bob@example.com")
	p := seedPeriod(t, ctx, "880e8400-e29b-41d4-a716-446655440001",
		"660e8400-e29b-41d4-a716-446655440001", "2026 上期", "H1", "2026-04-01", "2026-09-30")

	repo := repository.NewPgObjectiveRepository(testhelper.Pool)

	// Alice が owner
	mustSaveObjective(t, ctx, repo, buildObjective(t,
		"990e8400-e29b-41d4-a716-446655440001", p.ID().Value(), alice.ID().Value(), "Alice OKR-1", 0))
	mustSaveObjective(t, ctx, repo, buildObjective(t,
		"990e8400-e29b-41d4-a716-446655440002", p.ID().Value(), alice.ID().Value(), "Alice OKR-2", 1))
	// Bob が owner
	mustSaveObjective(t, ctx, repo, buildObjective(t,
		"990e8400-e29b-41d4-a716-446655440003", p.ID().Value(), bob.ID().Value(), "Bob OKR-1", 0))

	objs, err := repo.FindByOwnerID(ctx, alice.ID())
	if err != nil {
		t.Fatalf("FindByOwnerID: %v", err)
	}
	if len(objs) != 2 {
		t.Errorf("FindByOwnerID len = %d, want 2", len(objs))
	}
}

// ─── Remove ──────────────────────────────────────────────────────────────────

func TestPgObjectiveRepository_Remove(t *testing.T) {
	t.Cleanup(func() { testhelper.TruncateAll(t, testhelper.Pool) })
	ctx := context.Background()
	seedTeam(t, ctx, "660e8400-e29b-41d4-a716-446655440001", "Alpha")
	alice := seedUser(t, ctx, "550e8400-e29b-41d4-a716-446655440001", "Alice", "alice@example.com")
	p := seedPeriod(t, ctx, "880e8400-e29b-41d4-a716-446655440001",
		"660e8400-e29b-41d4-a716-446655440001", "2026 上期", "H1", "2026-04-01", "2026-09-30")

	repo := repository.NewPgObjectiveRepository(testhelper.Pool)
	o := buildObjective(t,
		"990e8400-e29b-41d4-a716-446655440001", p.ID().Value(), alice.ID().Value(), "OKR-1", 0)
	mustSaveObjective(t, ctx, repo, o)

	if err := repo.Remove(ctx, o.ID()); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	got, err := repo.FindByID(ctx, o.ID())
	if err != nil {
		t.Fatalf("FindByID after Remove: %v", err)
	}
	if got != nil {
		t.Error("Remove: objective still exists after Remove")
	}
}

func TestPgObjectiveRepository_Remove_Idempotent(t *testing.T) {
	t.Cleanup(func() { testhelper.TruncateAll(t, testhelper.Pool) })
	ctx := context.Background()
	repo := repository.NewPgObjectiveRepository(testhelper.Pool)

	id := mustObjectiveId(t, "990e8400-e29b-41d4-a716-446655440099")
	if err := repo.Remove(ctx, id); err != nil {
		t.Errorf("Remove (non-existent): want no error, got %v", err)
	}
}
