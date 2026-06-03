
[package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/keyresult"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/objective"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/infrastructure/repository"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/internal/testhelper"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

func mustKeyResultId(t *testing.T, v string) keyresult.KeyResultId {
	t.Helper()
	id, err := keyresult.NewKeyResultId(v)
	if err != nil {
		t.Fatalf("NewKeyResultId(%q): %v", v, err)
	}
	return id
}

func mustKrProgressLogId(t *testing.T, v string) keyresult.KrProgressLogId {
	t.Helper()
	id, err := keyresult.NewKrProgressLogId(v)
	if err != nil {
		t.Fatalf("NewKrProgressLogId(%q): %v", v, err)
	}
	return id
}

func buildNumericKR(t *testing.T, rawID, objectiveRawID, ownerRawID, title string, target float64, order int) keyresult.KeyResult {
	t.Helper()
	id := mustKeyResultId(t, rawID)
	oid, err := objective.NewObjectiveId(objectiveRawID)
	if err != nil {
		t.Fatalf("NewObjectiveId: %v", err)
	}
	uid := mustUserId(t, ownerRawID)
	kr, err := keyresult.NewNumericKeyResult(id, oid, uid, title, target, order, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("NewNumericKeyResult: %v", err)
	}
	return kr
}

func buildCheckboxKR(t *testing.T, rawID, objectiveRawID, ownerRawID, title string, order int) keyresult.KeyResult {
	t.Helper()
	id := mustKeyResultId(t, rawID)
	oid, err := objective.NewObjectiveId(objectiveRawID)
	if err != nil {
		t.Fatalf("NewObjectiveId: %v", err)
	}
	uid := mustUserId(t, ownerRawID)
	kr, err := keyresult.NewCheckboxKeyResult(id, oid, uid, title, order, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("NewCheckboxKeyResult: %v", err)
	}
	return kr
}

func mustSaveKR(t *testing.T, ctx context.Context, repo interface {
	Save(context.Context, keyresult.KeyResult) error
}, kr keyresult.KeyResult) {
	t.Helper()
	if err := repo.Save(ctx, kr); err != nil {
		t.Fatalf("Save(%v): %v", kr.ID().Value(), err)
	}
}

// seedObjective saves an objective and returns it.
func seedObjective(t *testing.T, ctx context.Context, rawID, periodRawID, ownerRawID, title string) objective.Objective {
	t.Helper()
	o := buildObjective(t, rawID, periodRawID, ownerRawID, title, 0)
	repo := repository.NewPgObjectiveRepository(testhelper.Pool)
	if err := repo.Save(ctx, o); err != nil {
		t.Fatalf("seedObjective Save: %v", err)
	}
	return o
}

// ─── Save / FindByID ─────────────────────────────────────────────────────────

func TestPgKeyResultRepository_Save_and_FindByID_Numeric(t *testing.T) {
	t.Cleanup(func() { testhelper.TruncateAll(t, testhelper.Pool) })
	ctx := context.Background()
	seedTeam(t, ctx, "660e8400-e29b-41d4-a716-446655440001", "Alpha")
	alice := seedUser(t, ctx, "550e8400-e29b-41d4-a716-446655440001", "Alice", "alice@example.com")
	p := seedPeriod(t, ctx, "880e8400-e29b-41d4-a716-446655440001",
		"660e8400-e29b-41d4-a716-446655440001", "2026 上期", "H1", "2026-04-01", "2026-09-30")
	o := seedObjective(t, ctx,
		"990e8400-e29b-41d4-a716-446655440001", p.ID().Value(), alice.ID().Value(), "OKR-1")

	repo := repository.NewPgKeyResultRepository(testhelper.Pool)
	kr := buildNumericKR(t,
		"aa0e8400-e29b-41d4-a716-446655440001",
		o.ID().Value(), alice.ID().Value(),
		"売上 150%達成", 150.0, 0,
	)
	mustSaveKR(t, ctx, repo, kr)

	got, err := repo.FindByID(ctx, kr.ID())
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got == nil {
		t.Fatal("FindByID: got nil, want key result")
	}
	if got.Title() != "売上 150%達成" {
		t.Errorf("Title = %q, want %q", got.Title(), "売上 150%達成")
	}
	if got.KrType() != keyresult.KrTypeNumeric {
		t.Errorf("KrType = %v, want numeric", got.KrType())
	}
	if got.TargetValue() == nil || *got.TargetValue() != 150.0 {
		t.Errorf("TargetValue = %v, want 150.0", got.TargetValue())
	}
}

func TestPgKeyResultRepository_Save_and_FindByID_Checkbox(t *testing.T) {
	t.Cleanup(func() { testhelper.TruncateAll(t, testhelper.Pool) })
	ctx := context.Background()
	seedTeam(t, ctx, "660e8400-e29b-41d4-a716-446655440001", "Alpha")
	alice := seedUser(t, ctx, "550e8400-e29b-41d4-a716-446655440001", "Alice", "alice@example.com")
	p := seedPeriod(t, ctx, "880e8400-e29b-41d4-a716-446655440001",
		"660e8400-e29b-41d4-a716-446655440001", "2026 上期", "H1", "2026-04-01", "2026-09-30")
	o := seedObjective(t, ctx,
		"990e8400-e29b-41d4-a716-446655440001", p.ID().Value(), alice.ID().Value(), "OKR-1")

	repo := repository.NewPgKeyResultRepository(testhelper.Pool)
	kr := buildCheckboxKR(t,
		"aa0e8400-e29b-41d4-a716-446655440001",
		o.ID().Value(), alice.ID().Value(),
		"新機能リリース完了", 0,
	)
	mustSaveKR(t, ctx, repo, kr)

	got, err := repo.FindByID(ctx, kr.ID())
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got == nil {
		t.Fatal("FindByID: got nil, want key result")
	}
	if got.KrType() != keyresult.KrTypeCheckbox {
		t.Errorf("KrType = %v, want checkbox", got.KrType())
	}
}

func TestPgKeyResultRepository_FindByID_NotFound(t *testing.T) {
	t.Cleanup(func() { testhelper.TruncateAll(t, testhelper.Pool) })
	ctx := context.Background()
	repo := repository.NewPgKeyResultRepository(testhelper.Pool)

	id := mustKeyResultId(t, "aa0e8400-e29b-41d4-a716-446655440099")
	got, err := repo.FindByID(ctx, id)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got != nil {
		t.Errorf("FindByID: got %v, want nil", got)
	}
}

// ─── Save + KrProgressLog ────────────────────────────────────────────────────

func TestPgKeyResultRepository_Save_WithProgressLog(t *testing.T) {
	t.Cleanup(func() { testhelper.TruncateAll(t, testhelper.Pool) })
	ctx := context.Background()
	seedTeam(t, ctx, "660e8400-e29b-41d4-a716-446655440001", "Alpha")
	alice := seedUser(t, ctx, "550e8400-e29b-41d4-a716-446655440001", "Alice", "alice@example.com")
	p := seedPeriod(t, ctx, "880e8400-e29b-41d4-a716-446655440001",
		"660e8400-e29b-41d4-a716-446655440001", "2026 上期", "H1", "2026-04-01", "2026-09-30")
	o := seedObjective(t, ctx,
		"990e8400-e29b-41d4-a716-446655440001", p.ID().Value(), alice.ID().Value(), "OKR-1")

	repo := repository.NewPgKeyResultRepository(testhelper.Pool)

	// 初回 Save（ProgressLog なし）
	kr := buildNumericKR(t,
		"aa0e8400-e29b-41d4-a716-446655440001",
		o.ID().Value(), alice.ID().Value(),
		"売上 150%達成", 150.0, 0,
	)
	mustSaveKR(t, ctx, repo, kr)

	// ProgressLog を追加して再 Save（append-only）
	logID := mustKrProgressLogId(t, "bb0e8400-e29b-41d4-a716-446655440001")
	note := "第1四半期終了時点"
	log := keyresult.NewNumericProgressLog(
		logID, kr.ID(), alice.ID(), 75.0, &note, time.Now(),
	)
	if err := kr.UpdateNumericProgress(log); err != nil {
		t.Fatalf("UpdateNumericProgress: %v", err)
	}
	mustSaveKR(t, ctx, repo, kr)

	got, err := repo.FindByID(ctx, kr.ID())
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got == nil {
		t.Fatal("FindByID: got nil")
	}
	if len(got.ProgressLogs()) != 1 {
		t.Errorf("ProgressLogs len = %d, want 1", len(got.ProgressLogs()))
	}
	if got.CurrentValue() == nil || *got.CurrentValue() != 75.0 {
		t.Errorf("CurrentValue = %v, want 75.0", got.CurrentValue())
	}
}

// ─── FindByObjectiveID ───────────────────────────────────────────────────────

func TestPgKeyResultRepository_FindByObjectiveID(t *testing.T) {
	t.Cleanup(func() { testhelper.TruncateAll(t, testhelper.Pool) })
	ctx := context.Background()
	seedTeam(t, ctx, "660e8400-e29b-41d4-a716-446655440001", "Alpha")
	alice := seedUser(t, ctx, "550e8400-e29b-41d4-a716-446655440001", "Alice", "alice@example.com")
	p := seedPeriod(t, ctx, "880e8400-e29b-41d4-a716-446655440001",
		"660e8400-e29b-41d4-a716-446655440001", "2026 上期", "H1", "2026-04-01", "2026-09-30")
	o1 := seedObjective(t, ctx,
		"990e8400-e29b-41d4-a716-446655440001", p.ID().Value(), alice.ID().Value(), "OKR-1")
	o2 := seedObjective(t, ctx,
		"990e8400-e29b-41d4-a716-446655440002", p.ID().Value(), alice.ID().Value(), "OKR-2")

	repo := repository.NewPgKeyResultRepository(testhelper.Pool)

	// o1 に 2 件
	mustSaveKR(t, ctx, repo, buildNumericKR(t,
		"aa0e8400-e29b-41d4-a716-446655440001", o1.ID().Value(), alice.ID().Value(), "KR-1", 100.0, 0))
	mustSaveKR(t, ctx, repo, buildCheckboxKR(t,
		"aa0e8400-e29b-41d4-a716-446655440002", o1.ID().Value(), alice.ID().Value(), "KR-2", 1))
	// o2 に 1 件
	mustSaveKR(t, ctx, repo, buildNumericKR(t,
		"aa0e8400-e29b-41d4-a716-446655440003", o2.ID().Value(), alice.ID().Value(), "KR-3", 50.0, 0))

	krs, err := repo.FindByObjectiveID(ctx, o1.ID())
	if err != nil {
		t.Fatalf("FindByObjectiveID: %v", err)
	}
	if len(krs) != 2 {
		t.Errorf("FindByObjectiveID len = %d, want 2", len(krs))
	}
}

// ─── Remove + CASCADE 確認 ───────────────────────────────────────────────────

func TestPgKeyResultRepository_Remove(t *testing.T) {
	t.Cleanup(func() { testhelper.TruncateAll(t, testhelper.Pool) })
	ctx := context.Background()
	seedTeam(t, ctx, "660e8400-e29b-41d4-a716-446655440001", "Alpha")
	alice := seedUser(t, ctx, "550e8400-e29b-41d4-a716-446655440001", "Alice", "alice@example.com")
	p := seedPeriod(t, ctx, "880e8400-e29b-41d4-a716-446655440001",
		"660e8400-e29b-41d4-a716-446655440001", "2026 上期", "H1", "2026-04-01", "2026-09-30")
	o := seedObjective(t, ctx,
		"990e8400-e29b-41d4-a716-446655440001", p.ID().Value(), alice.ID().Value(), "OKR-1")

	repo := repository.NewPgKeyResultRepository(testhelper.Pool)

	// KR + ProgressLog を保存
	kr := buildNumericKR(t,
		"aa0e8400-e29b-41d4-a716-446655440001",
		o.ID().Value(), alice.ID().Value(),
		"売上 150%達成", 150.0, 0,
	)
	mustSaveKR(t, ctx, repo, kr)

	logID := mustKrProgressLogId(t, "bb0e8400-e29b-41d4-a716-446655440001")
	log := keyresult.NewNumericProgressLog(logID, kr.ID(), alice.ID(), 50.0, nil, time.Now())
	if err := kr.UpdateNumericProgress(log); err != nil {
		t.Fatalf("UpdateNumericProgress: %v", err)
	}
	mustSaveKR(t, ctx, repo, kr)

	// KR を削除
	if err := repo.Remove(ctx, kr.ID()); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	// KR が消えている
	got, err := repo.FindByID(ctx, kr.ID())
	if err != nil {
		t.Fatalf("FindByID after Remove: %v", err)
	}
	if got != nil {
		t.Error("Remove: key result still exists after Remove")
	}
}

func TestPgKeyResultRepository_Remove_Idempotent(t *testing.T) {
	t.Cleanup(func() { testhelper.TruncateAll(t, testhelper.Pool) })
	ctx := context.Background()
	repo := repository.NewPgKeyResultRepository(testhelper.Pool)

	id := mustKeyResultId(t, "aa0e8400-e29b-41d4-a716-446655440099")
	if err := repo.Remove(ctx, id); err != nil {
		t.Errorf("Remove (non-existent): want no error, got %v", err)
	}
}
