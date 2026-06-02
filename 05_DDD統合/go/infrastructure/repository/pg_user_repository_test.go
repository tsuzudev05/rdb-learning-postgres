package repository_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/user"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/infrastructure/repository"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/internal/testhelper"
)

// ─── TestMain ────────────────────────────────────────────────────────────────

func TestMain(m *testing.M) {
	ctx := context.Background()
	pool, teardown := testhelper.SetupPostgres(ctx)
	defer teardown()
	testhelper.Pool = pool
	os.Exit(m.Run())
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func mustUserId(t *testing.T, v string) user.UserId {
	t.Helper()
	id, err := user.NewUserId(v)
	if err != nil {
		t.Fatalf("NewUserId(%q): %v", v, err)
	}
	return id
}

func mustEmail(t *testing.T, v string) user.Email {
	t.Helper()
	e, err := user.NewEmail(v)
	if err != nil {
		t.Fatalf("NewEmail(%q): %v", v, err)
	}
	return e
}

func buildUser(t *testing.T, rawID, name, email string) user.User {
	t.Helper()
	id := mustUserId(t, rawID)
	e := mustEmail(t, email)
	u, err := user.NewUser(id, name, e, "hash_"+name, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("NewUser: %v", err)
	}
	return u
}

// mustSaveUser saves a user and fails the test if Save returns an error.
func mustSaveUser(t *testing.T, ctx context.Context, repo interface {
	Save(context.Context, user.User) error
}, u user.User) {
	t.Helper()
	if err := repo.Save(ctx, u); err != nil {
		t.Fatalf("Save(%v): %v", u.ID().Value(), err)
	}
}

// ─── Save / FindByID ─────────────────────────────────────────────────────────

func TestPgUserRepository_Save_and_FindByID(t *testing.T) {
	t.Cleanup(func() { testhelper.TruncateAll(t, testhelper.Pool) })
	ctx := context.Background()
	repo := repository.NewPgUserRepository(testhelper.Pool)

	u := buildUser(t, "550e8400-e29b-41d4-a716-446655440000", "Alice", "alice@example.com")
	mustSaveUser(t, ctx, repo, u)

	got, err := repo.FindByID(ctx, u.ID())
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got == nil {
		t.Fatal("FindByID: got nil, want user")
	}
	if got.Email().Value() != "alice@example.com" {
		t.Errorf("Email = %q, want %q", got.Email().Value(), "alice@example.com")
	}
	if got.Name() != "Alice" {
		t.Errorf("Name = %q, want %q", got.Name(), "Alice")
	}
}

func TestPgUserRepository_FindByID_NotFound(t *testing.T) {
	t.Cleanup(func() { testhelper.TruncateAll(t, testhelper.Pool) })
	ctx := context.Background()
	repo := repository.NewPgUserRepository(testhelper.Pool)

	id := mustUserId(t, "550e8400-e29b-41d4-a716-446655440001")
	got, err := repo.FindByID(ctx, id)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got != nil {
		t.Errorf("FindByID: got %v, want nil", got)
	}
}

// ─── Save (upsert) ───────────────────────────────────────────────────────────

func TestPgUserRepository_Save_Upsert(t *testing.T) {
	t.Cleanup(func() { testhelper.TruncateAll(t, testhelper.Pool) })
	ctx := context.Background()
	repo := repository.NewPgUserRepository(testhelper.Pool)

	u := buildUser(t, "550e8400-e29b-41d4-a716-446655440000", "Alice", "alice@example.com")
	mustSaveUser(t, ctx, repo, u)

	// 名前を変えて再 Save（upsert）
	updated := buildUser(t, "550e8400-e29b-41d4-a716-446655440000", "Alice Updated", "alice@example.com")
	mustSaveUser(t, ctx, repo, updated)

	got, err := repo.FindByID(ctx, u.ID())
	if err != nil {
		t.Fatalf("FindByID after upsert: %v", err)
	}
	if got == nil {
		t.Fatal("FindByID after upsert: got nil")
	}
	if got.Name() != "Alice Updated" {
		t.Errorf("Name after upsert = %q, want %q", got.Name(), "Alice Updated")
	}
}

// ─── FindByEmail ─────────────────────────────────────────────────────────────

func TestPgUserRepository_FindByEmail(t *testing.T) {
	t.Cleanup(func() { testhelper.TruncateAll(t, testhelper.Pool) })
	ctx := context.Background()
	repo := repository.NewPgUserRepository(testhelper.Pool)

	u := buildUser(t, "550e8400-e29b-41d4-a716-446655440000", "Alice", "alice@example.com")
	mustSaveUser(t, ctx, repo, u)

	email := mustEmail(t, "alice@example.com")
	got, err := repo.FindByEmail(ctx, email)
	if err != nil {
		t.Fatalf("FindByEmail: %v", err)
	}
	if got == nil || got.ID().Value() != u.ID().Value() {
		t.Errorf("FindByEmail: got %v, want %v", got, u.ID())
	}
}

func TestPgUserRepository_FindByEmail_NotFound(t *testing.T) {
	t.Cleanup(func() { testhelper.TruncateAll(t, testhelper.Pool) })
	ctx := context.Background()
	repo := repository.NewPgUserRepository(testhelper.Pool)

	email := mustEmail(t, "nobody@example.com")
	got, err := repo.FindByEmail(ctx, email)
	if err != nil {
		t.Fatalf("FindByEmail: %v", err)
	}
	if got != nil {
		t.Errorf("FindByEmail: got %v, want nil", got)
	}
}

// ─── FindAll ─────────────────────────────────────────────────────────────────

func TestPgUserRepository_FindAll_Empty(t *testing.T) {
	t.Cleanup(func() { testhelper.TruncateAll(t, testhelper.Pool) })
	ctx := context.Background()
	repo := repository.NewPgUserRepository(testhelper.Pool)

	all, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("FindAll len = %d, want 0", len(all))
	}
}

func TestPgUserRepository_FindAll(t *testing.T) {
	t.Cleanup(func() { testhelper.TruncateAll(t, testhelper.Pool) })
	ctx := context.Background()
	repo := repository.NewPgUserRepository(testhelper.Pool)

	seeds := []user.User{
		buildUser(t, "550e8400-e29b-41d4-a716-446655440001", "Alice", "alice@example.com"),
		buildUser(t, "550e8400-e29b-41d4-a716-446655440002", "Bob", "bob@example.com"),
		buildUser(t, "550e8400-e29b-41d4-a716-446655440003", "Carol", "carol@example.com"),
	}
	for _, u := range seeds {
		mustSaveUser(t, ctx, repo, u)
	}

	all, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("FindAll len = %d, want 3", len(all))
	}
}

// ─── Remove ──────────────────────────────────────────────────────────────────

func TestPgUserRepository_Remove(t *testing.T) {
	t.Cleanup(func() { testhelper.TruncateAll(t, testhelper.Pool) })
	ctx := context.Background()
	repo := repository.NewPgUserRepository(testhelper.Pool)

	u := buildUser(t, "550e8400-e29b-41d4-a716-446655440000", "Alice", "alice@example.com")
	mustSaveUser(t, ctx, repo, u)

	if err := repo.Remove(ctx, u.ID()); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	got, err := repo.FindByID(ctx, u.ID())
	if err != nil {
		t.Fatalf("FindByID after Remove: %v", err)
	}
	if got != nil {
		t.Error("Remove: user still exists after Remove")
	}
}

func TestPgUserRepository_Remove_Idempotent(t *testing.T) {
	t.Cleanup(func() { testhelper.TruncateAll(t, testhelper.Pool) })
	ctx := context.Background()
	repo := repository.NewPgUserRepository(testhelper.Pool)

	id := mustUserId(t, "550e8400-e29b-41d4-a716-446655440000")
	// 存在しない ID の Remove はエラーにならない
	if err := repo.Remove(ctx, id); err != nil {
		t.Errorf("Remove (non-existent): want no error, got %v", err)
	}
}
