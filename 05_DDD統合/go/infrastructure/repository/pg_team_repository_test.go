package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/team"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/user"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/infrastructure/repository"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/internal/testhelper"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

func mustTeamId(t *testing.T, v string) team.TeamId {
	t.Helper()
	id, err := team.NewTeamId(v)
	if err != nil {
		t.Fatalf("NewTeamId(%q): %v", v, err)
	}
	return id
}

func mustTeamMemberId(t *testing.T, v string) team.TeamMemberId {
	t.Helper()
	id, err := team.NewTeamMemberId(v)
	if err != nil {
		t.Fatalf("NewTeamMemberId(%q): %v", v, err)
	}
	return id
}

// buildTeam builds a Team with no members.
func buildTeam(t *testing.T, rawID, name string) team.Team {
	t.Helper()
	id := mustTeamId(t, rawID)
	desc := "テストチーム: " + name
	tm, err := team.NewTeam(id, name, &desc, nil, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("NewTeam: %v", err)
	}
	return tm
}

// buildMember builds a TeamMember.
func buildMember(t *testing.T, memberID, teamID string, uid user.UserId, role team.Role) team.TeamMember {
	t.Helper()
	mid := mustTeamMemberId(t, memberID)
	tid := mustTeamId(t, teamID)
	m, err := team.NewTeamMember(mid, tid, uid, role, time.Now())
	if err != nil {
		t.Fatalf("NewTeamMember: %v", err)
	}
	return m
}

// seedUser saves a user and returns it.
func seedUser(t *testing.T, ctx context.Context, rawID, name, email string) user.User {
	t.Helper()
	u := buildUser(t, rawID, name, email)
	userRepo := repository.NewPgUserRepository(testhelper.Pool)
	if err := userRepo.Save(ctx, u); err != nil {
		t.Fatalf("seedUser Save: %v", err)
	}
	return u
}

// mustSaveTeam saves a team and fails the test if Save returns an error.
func mustSaveTeam(t *testing.T, ctx context.Context, repo interface {
	Save(context.Context, team.Team) error
}, tm team.Team) {
	t.Helper()
	if err := repo.Save(ctx, tm); err != nil {
		t.Fatalf("Save(%v): %v", tm.ID().Value(), err)
	}
}

// ─── Save / FindByID ─────────────────────────────────────────────────────────

func TestPgTeamRepository_Save_and_FindByID(t *testing.T) {
	t.Cleanup(func() { testhelper.TruncateAll(t, testhelper.Pool) })
	ctx := context.Background()
	repo := repository.NewPgTeamRepository(testhelper.Pool)

	tm := buildTeam(t, "660e8400-e29b-41d4-a716-446655440000", "Alpha")
	mustSaveTeam(t, ctx, repo, tm)

	got, err := repo.FindByID(ctx, tm.ID())
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got == nil {
		t.Fatal("FindByID: got nil, want team")
	}
	if got.Name() != "Alpha" {
		t.Errorf("Name = %q, want %q", got.Name(), "Alpha")
	}
}

func TestPgTeamRepository_FindByID_NotFound(t *testing.T) {
	t.Cleanup(func() { testhelper.TruncateAll(t, testhelper.Pool) })
	ctx := context.Background()
	repo := repository.NewPgTeamRepository(testhelper.Pool)

	id := mustTeamId(t, "660e8400-e29b-41d4-a716-446655440099")
	got, err := repo.FindByID(ctx, id)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got != nil {
		t.Errorf("FindByID: got %v, want nil", got)
	}
}

// ─── Save (upsert) ───────────────────────────────────────────────────────────

func TestPgTeamRepository_Save_Upsert(t *testing.T) {
	t.Cleanup(func() { testhelper.TruncateAll(t, testhelper.Pool) })
	ctx := context.Background()
	repo := repository.NewPgTeamRepository(testhelper.Pool)

	tm := buildTeam(t, "660e8400-e29b-41d4-a716-446655440000", "Alpha")
	mustSaveTeam(t, ctx, repo, tm)

	updated := buildTeam(t, "660e8400-e29b-41d4-a716-446655440000", "Alpha Updated")
	mustSaveTeam(t, ctx, repo, updated)

	got, err := repo.FindByID(ctx, tm.ID())
	if err != nil {
		t.Fatalf("FindByID after upsert: %v", err)
	}
	if got == nil {
		t.Fatal("FindByID after upsert: got nil")
	}
	if got.Name() != "Alpha Updated" {
		t.Errorf("Name after upsert = %q, want %q", got.Name(), "Alpha Updated")
	}
}

// ─── TeamMember の保存・full-replace ─────────────────────────────────────────

func TestPgTeamRepository_Save_WithMembers(t *testing.T) {
	t.Cleanup(func() { testhelper.TruncateAll(t, testhelper.Pool) })
	ctx := context.Background()
	repo := repository.NewPgTeamRepository(testhelper.Pool)

	alice := seedUser(t, ctx, "550e8400-e29b-41d4-a716-446655440001", "Alice", "alice@example.com")
	bob   := seedUser(t, ctx, "550e8400-e29b-41d4-a716-446655440002", "Bob", "bob@example.com")

	tm := buildTeam(t, "660e8400-e29b-41d4-a716-446655440000", "Alpha")
	tm.AddMember(buildMember(t,
		"770e8400-e29b-41d4-a716-446655440001",
		"660e8400-e29b-41d4-a716-446655440000",
		alice.ID(), team.Admin(),
	))
	tm.AddMember(buildMember(t,
		"770e8400-e29b-41d4-a716-446655440002",
		"660e8400-e29b-41d4-a716-446655440000",
		bob.ID(), team.Member(),
	))
	mustSaveTeam(t, ctx, repo, tm)

	got, err := repo.FindByID(ctx, tm.ID())
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got == nil {
		t.Fatal("FindByID: got nil")
	}
	if len(got.Members()) != 2 {
		t.Errorf("members count = %d, want 2", len(got.Members()))
	}
}

func TestPgTeamRepository_Save_FullReplace_Members(t *testing.T) {
	t.Cleanup(func() { testhelper.TruncateAll(t, testhelper.Pool) })
	ctx := context.Background()
	repo := repository.NewPgTeamRepository(testhelper.Pool)

	alice := seedUser(t, ctx, "550e8400-e29b-41d4-a716-446655440001", "Alice", "alice@example.com")
	bob   := seedUser(t, ctx, "550e8400-e29b-41d4-a716-446655440002", "Bob", "bob@example.com")

	// 初回: Alice だけ
	tm := buildTeam(t, "660e8400-e29b-41d4-a716-446655440000", "Alpha")
	tm.AddMember(buildMember(t,
		"770e8400-e29b-41d4-a716-446655440001",
		"660e8400-e29b-41d4-a716-446655440000",
		alice.ID(), team.Admin(),
	))
	mustSaveTeam(t, ctx, repo, tm)

	// 2回目: Bob だけ（full-replace → Alice が消える）
	tm2 := buildTeam(t, "660e8400-e29b-41d4-a716-446655440000", "Alpha")
	tm2.AddMember(buildMember(t,
		"770e8400-e29b-41d4-a716-446655440002",
		"660e8400-e29b-41d4-a716-446655440000",
		bob.ID(), team.Member(),
	))
	mustSaveTeam(t, ctx, repo, tm2)

	got, err := repo.FindByID(ctx, tm.ID())
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got == nil {
		t.Fatal("FindByID: got nil")
	}
	if len(got.Members()) != 1 {
		t.Errorf("members count = %d, want 1 (full-replace)", len(got.Members()))
	}
	if got.Members()[0].UserID().Value() != bob.ID().Value() {
		t.Errorf("remaining member = %v, want Bob", got.Members()[0].UserID())
	}
}

// ─── FindByUserID ────────────────────────────────────────────────────────────

func TestPgTeamRepository_FindByUserID(t *testing.T) {
	t.Cleanup(func() { testhelper.TruncateAll(t, testhelper.Pool) })
	ctx := context.Background()
	repo := repository.NewPgTeamRepository(testhelper.Pool)

	alice := seedUser(t, ctx, "550e8400-e29b-41d4-a716-446655440001", "Alice", "alice@example.com")

	// Alpha に Alice を追加
	alpha := buildTeam(t, "660e8400-e29b-41d4-a716-446655440001", "Alpha")
	alpha.AddMember(buildMember(t,
		"770e8400-e29b-41d4-a716-446655440001",
		"660e8400-e29b-41d4-a716-446655440001",
		alice.ID(), team.Admin(),
	))
	mustSaveTeam(t, ctx, repo, alpha)

	// Beta には Alice なし
	beta := buildTeam(t, "660e8400-e29b-41d4-a716-446655440002", "Beta")
	mustSaveTeam(t, ctx, repo, beta)

	teams, err := repo.FindByUserID(ctx, alice.ID())
	if err != nil {
		t.Fatalf("FindByUserID: %v", err)
	}
	if len(teams) != 1 {
		t.Errorf("FindByUserID len = %d, want 1", len(teams))
	}
	if teams[0].ID().Value() != alpha.ID().Value() {
		t.Errorf("team = %v, want Alpha", teams[0].ID())
	}
}

// TestPgTeamRepository_FindByUserID_MultipleTeams confirms that a user
// belonging to more than one team gets all of them back.
func TestPgTeamRepository_FindByUserID_MultipleTeams(t *testing.T) {
	t.Cleanup(func() { testhelper.TruncateAll(t, testhelper.Pool) })
	ctx := context.Background()
	repo := repository.NewPgTeamRepository(testhelper.Pool)

	alice := seedUser(t, ctx, "550e8400-e29b-41d4-a716-446655440001", "Alice", "alice@example.com")

	for i, name := range []string{"Alpha", "Beta", "Gamma"} {
		ids := []string{
			"660e8400-e29b-41d4-a716-446655440001",
			"660e8400-e29b-41d4-a716-446655440002",
			"660e8400-e29b-41d4-a716-446655440003",
		}
		memberIDs := []string{
			"770e8400-e29b-41d4-a716-446655440001",
			"770e8400-e29b-41d4-a716-446655440002",
			"770e8400-e29b-41d4-a716-446655440003",
		}
		tm := buildTeam(t, ids[i], name)
		tm.AddMember(buildMember(t, memberIDs[i], ids[i], alice.ID(), team.Member()))
		mustSaveTeam(t, ctx, repo, tm)
	}

	teams, err := repo.FindByUserID(ctx, alice.ID())
	if err != nil {
		t.Fatalf("FindByUserID: %v", err)
	}
	if len(teams) != 3 {
		t.Errorf("FindByUserID len = %d, want 3", len(teams))
	}
}

// ─── Remove (CASCADE 確認) ───────────────────────────────────────────────────

func TestPgTeamRepository_Remove_CascadesMembers(t *testing.T) {
	t.Cleanup(func() { testhelper.TruncateAll(t, testhelper.Pool) })
	ctx := context.Background()
	repo := repository.NewPgTeamRepository(testhelper.Pool)

	alice := seedUser(t, ctx, "550e8400-e29b-41d4-a716-446655440001", "Alice", "alice@example.com")

	tm := buildTeam(t, "660e8400-e29b-41d4-a716-446655440000", "Alpha")
	tm.AddMember(buildMember(t,
		"770e8400-e29b-41d4-a716-446655440001",
		"660e8400-e29b-41d4-a716-446655440000",
		alice.ID(), team.Admin(),
	))
	mustSaveTeam(t, ctx, repo, tm)

	if err := repo.Remove(ctx, tm.ID()); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	// Team 自体が消えている
	got, err := repo.FindByID(ctx, tm.ID())
	if err != nil {
		t.Fatalf("FindByID after Remove: %v", err)
	}
	if got != nil {
		t.Error("Remove: team still exists after Remove")
	}

	// Alice が所属するチームも 0 件になる（CASCADE DELETE 確認）
	teams, err := repo.FindByUserID(ctx, alice.ID())
	if err != nil {
		t.Fatalf("FindByUserID after Remove: %v", err)
	}
	if len(teams) != 0 {
		t.Errorf("FindByUserID after team removed = %d, want 0", len(teams))
	}
}

func TestPgTeamRepository_Remove_Idempotent(t *testing.T) {
	t.Cleanup(func() { testhelper.TruncateAll(t, testhelper.Pool) })
	ctx := context.Background()
	repo := repository.NewPgTeamRepository(testhelper.Pool)

	id := mustTeamId(t, "660e8400-e29b-41d4-a716-446655440000")
	if err := repo.Remove(ctx, id); err != nil {
		t.Errorf("Remove (non-existent): want no error, got %v", err)
	}
}

// ─── FindAll ─────────────────────────────────────────────────────────────────

func TestPgTeamRepository_FindAll_Empty(t *testing.T) {
	t.Cleanup(func() { testhelper.TruncateAll(t, testhelper.Pool) })
	ctx := context.Background()
	repo := repository.NewPgTeamRepository(testhelper.Pool)

	all, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("FindAll len = %d, want 0", len(all))
	}
}

func TestPgTeamRepository_FindAll(t *testing.T) {
	t.Cleanup(func() { testhelper.TruncateAll(t, testhelper.Pool) })
	ctx := context.Background()
	repo := repository.NewPgTeamRepository(testhelper.Pool)

	ids := []string{
		"660e8400-e29b-41d4-a716-446655440001",
		"660e8400-e29b-41d4-a716-446655440002",
		"660e8400-e29b-41d4-a716-446655440003",
	}
	for i, name := range []string{"Alpha", "Beta", "Gamma"} {
		mustSaveTeam(t, ctx, repo, buildTeam(t, ids[i], name))
	}

	all, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("FindAll len = %d, want 3", len(all))
	}
}
