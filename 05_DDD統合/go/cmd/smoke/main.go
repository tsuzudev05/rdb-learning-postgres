// smoke — Go/pgx Repository スモークテスト
//
// DevContainer 内での実行:
//
//	cd /workspace/05_DDD統合/go
//	go run ./cmd/smoke
//
// 環境変数 DATABASE_URL が設定されていない場合は既定の接続文字列を使用する。
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/keyresult"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/objective"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/period"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/team"
	"github.com/tsuzudev05/rdb-learning-postgres/okr/domain/model/user"
	infrarepo "github.com/tsuzudev05/rdb-learning-postgres/okr/infrastructure/repository"
)

const defaultDSN = "postgresql://postgres:pass@postgres:5432/learning"

func connString() string {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}
	return defaultDSN
}

func main() {
	ctx := context.Background()

	fmt.Println("========================================")
	fmt.Println(" OKR Repository スモークテスト (Go/pgx)")
	fmt.Println("========================================")

	pool, err := pgxpool.New(ctx, connString())
	if err != nil {
		log.Fatalf("DB接続失敗: %v", err)
	}
	defer pool.Close()

	testConnection(ctx, pool)      // [1]
	testUserRepository(ctx, pool)  // [2]-[7]
	testTeamRepository(ctx, pool)  // [8]-[12]
	testPeriodRepository(ctx, pool) // [13]-[17]
	testObjectiveRepository(ctx, pool) // [18]-[21]
	testKeyResultRepository(ctx, pool) // [22]-[26]

	fmt.Println("========================================")
	fmt.Println(" ✅ 全テスト PASSED")
	fmt.Println("========================================")
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func must(label string, err error) {
	if err != nil {
		log.Fatalf("❌ %s: %v", label, err)
	}
}

// ─── [1] DB 接続確認 ──────────────────────────────────────────────────────────

func testConnection(ctx context.Context, pool *pgxpool.Pool) {
	fmt.Print("[1] DB 接続確認... ")
	var dbName string
	must("DB接続確認", pool.QueryRow(ctx, "SELECT current_database()").Scan(&dbName))
	fmt.Printf("✅  DB=%s\n", dbName)
}

// ─── [2-7] UserRepository ────────────────────────────────────────────────────

func testUserRepository(ctx context.Context, pool *pgxpool.Pool) {
	repo := infrarepo.NewPgUserRepository(pool)

	const uuid = "22222222-2222-4222-a222-222222222222"
	id, _ := user.NewUserId(uuid)
	email, _ := user.NewEmail("gosmoke@example.com")

	must("remove(pre-cleanup)", repo.Remove(ctx, id))

	fmt.Print("[2] UserRepository::Save (新規)... ")
	u, err := user.NewUser(id, "スモーク太郎Go", email, "hash_go_xxx", time.Time{}, time.Time{})
	must("NewUser", err)
	must("Save(new)", repo.Save(ctx, u))
	fmt.Println("✅")

	fmt.Print("[3] UserRepository::FindByID... ")
	found, err := repo.FindByID(ctx, id)
	must("FindByID", err)
	if found == nil || found.Name() != "スモーク太郎Go" {
		log.Fatal("❌ FindByID: 名前が一致しません")
	}
	fmt.Printf("✅  name=%s\n", found.Name())

	fmt.Print("[4] UserRepository::Save (更新)... ")
	upd, _ := user.NewUser(id, "スモーク太郎Go更新済み", email, "hash_go_yyy", time.Time{}, time.Time{})
	must("Save(update)", repo.Save(ctx, upd))
	conf, _ := repo.FindByID(ctx, id)
	if conf == nil || conf.Name() != "スモーク太郎Go更新済み" {
		log.Fatal("❌ Save(update): 更新が反映されていません")
	}
	fmt.Printf("✅  name=%s\n", conf.Name())

	fmt.Print("[5] UserRepository::FindByEmail... ")
	byEmail, err := repo.FindByEmail(ctx, email)
	must("FindByEmail", err)
	if byEmail == nil || !byEmail.ID().Equal(id) {
		log.Fatal("❌ FindByEmail: IDが一致しません")
	}
	fmt.Println("✅")

	fmt.Print("[6] UserRepository::FindAll... ")
	all, err := repo.FindAll(ctx)
	must("FindAll", err)
	if len(all) == 0 {
		log.Fatal("❌ FindAll: 結果が0件です")
	}
	fmt.Printf("✅  count=%d\n", len(all))

	fmt.Print("[7] UserRepository::Remove... ")
	must("Remove", repo.Remove(ctx, id))
	gone, _ := repo.FindByID(ctx, id)
	if gone != nil {
		log.Fatal("❌ Remove: 削除後もユーザーが存在します")
	}
	fmt.Println("✅")
}

// ─── [8-12] TeamRepository ───────────────────────────────────────────────────

func testTeamRepository(ctx context.Context, pool *pgxpool.Pool) {
	userRepo := infrarepo.NewPgUserRepository(pool)
	teamRepo := infrarepo.NewPgTeamRepository(pool)

	userUUID   := "33333333-3333-4333-a333-333333333333"
	teamUUID   := "44444444-4444-4444-a444-444444444444"
	memberUUID := "55555555-5555-4555-a555-555555555555"

	uid, _ := user.NewUserId(userUUID)
	tid, _ := team.NewTeamId(teamUUID)
	mid, _ := team.NewTeamMemberId(memberUUID)

	// cleanup
	must("team remove(pre)", teamRepo.Remove(ctx, tid))
	must("user remove(pre)", userRepo.Remove(ctx, uid))

	// テスト用 User
	email, _ := user.NewEmail("team-go-smoke@example.com")
	u, _ := user.NewUser(uid, "チームテストユーザーGo", email, "hash", time.Time{}, time.Time{})
	must("user save for team", userRepo.Save(ctx, u))

	// Team 生成（メンバー付き）
	fmt.Print("[8] TeamRepository::Save (新規+メンバー)... ")
	t, err := team.NewTeam(tid, "Goスモークチーム", nil, nil, time.Time{}, time.Time{})
	must("NewTeam", err)
	m, err := team.NewTeamMember(mid, tid, uid, team.Admin(), time.Now())
	must("NewTeamMember", err)
	must("AddMember", t.AddMember(m))
	must("TeamRepo.Save", teamRepo.Save(ctx, t))
	fmt.Println("✅")

	fmt.Print("[9] TeamRepository::FindByID... ")
	found, err := teamRepo.FindByID(ctx, tid)
	must("FindByID", err)
	if found == nil || found.Name() != "Goスモークチーム" || len(found.Members()) != 1 {
		log.Fatalf("❌ FindByID: name=%v members=%d", found.Name(), len(found.Members()))
	}
	fmt.Printf("✅  name=%s  members=%d\n", found.Name(), len(found.Members()))

	fmt.Print("[10] TeamRepository::FindByUserID... ")
	byUser, err := teamRepo.FindByUserID(ctx, uid)
	must("FindByUserID", err)
	if len(byUser) == 0 {
		log.Fatal("❌ FindByUserID: 結果が0件です")
	}
	fmt.Printf("✅  count=%d\n", len(byUser))

	fmt.Print("[11] TeamRepository::FindAll... ")
	all, err := teamRepo.FindAll(ctx)
	must("FindAll", err)
	if len(all) == 0 {
		log.Fatal("❌ FindAll: 結果が0件です")
	}
	fmt.Printf("✅  count=%d\n", len(all))

	fmt.Print("[12] TeamRepository::Remove... ")
	must("Remove", teamRepo.Remove(ctx, tid))
	gone, _ := teamRepo.FindByID(ctx, tid)
	if gone != nil {
		log.Fatal("❌ Remove: 削除後もチームが存在します")
	}
	fmt.Println("✅")

	must("user remove(post)", userRepo.Remove(ctx, uid))
}

// ─── [13-17] PeriodRepository ────────────────────────────────────────────────

func testPeriodRepository(ctx context.Context, pool *pgxpool.Pool) {
	userRepo   := infrarepo.NewPgUserRepository(pool)
	teamRepo   := infrarepo.NewPgTeamRepository(pool)
	periodRepo := infrarepo.NewPgPeriodRepository(pool)

	userUUID   := "66666666-6666-4666-a666-666666666666"
	teamUUID   := "77777777-7777-4777-a777-777777777777"
	memberUUID := "88888888-8888-4888-a888-888888888888"
	periodUUID := "99999999-9999-4999-a999-999999999999"

	uid, _ := user.NewUserId(userUUID)
	tid, _ := team.NewTeamId(teamUUID)
	mid, _ := team.NewTeamMemberId(memberUUID)
	pid, _ := period.NewPeriodId(periodUUID)

	// cleanup
	must("period remove(pre)", periodRepo.Remove(ctx, pid))
	must("team remove(pre)",   teamRepo.Remove(ctx, tid))
	must("user remove(pre)",   userRepo.Remove(ctx, uid))

	// 前提データ
	email, _ := user.NewEmail("period-go-smoke@example.com")
	u, _ := user.NewUser(uid, "期間テストユーザーGo", email, "hash", time.Time{}, time.Time{})
	must("user save", userRepo.Save(ctx, u))

	t, _ := team.NewTeam(tid, "期間テストチームGo", nil, nil, time.Time{}, time.Time{})
	m, _ := team.NewTeamMember(mid, tid, uid, team.Admin(), time.Now())
	must("AddMember", t.AddMember(m))
	must("team save", teamRepo.Save(ctx, t))

	// Period 生成
	fmt.Print("[13] PeriodRepository::Save (新規)... ")
	h, _ := period.NewHalf("H1")
	dr, _ := period.NewDateRange("2025-01-01", "2025-06-30")
	p, err := period.NewPeriod(pid, tid, "2025-H1-Go", h, dr, time.Time{}, time.Time{})
	must("NewPeriod", err)
	must("PeriodRepo.Save", periodRepo.Save(ctx, p))
	fmt.Println("✅")

	fmt.Print("[14] PeriodRepository::FindByID... ")
	found, err := periodRepo.FindByID(ctx, pid)
	must("FindByID", err)
	if found == nil || found.Name() != "2025-H1-Go" {
		log.Fatalf("❌ FindByID: name=%v", found)
	}
	fmt.Printf("✅  name=%s  half=%s\n", found.Name(), found.Half())

	fmt.Print("[15] PeriodRepository::FindByTeamID... ")
	byTeam, err := periodRepo.FindByTeamID(ctx, tid)
	must("FindByTeamID", err)
	if len(byTeam) == 0 {
		log.Fatal("❌ FindByTeamID: 結果が0件です")
	}
	fmt.Printf("✅  count=%d\n", len(byTeam))

	fmt.Print("[16] PeriodRepository::FindAll... ")
	all, err := periodRepo.FindAll(ctx)
	must("FindAll", err)
	if len(all) == 0 {
		log.Fatal("❌ FindAll: 結果が0件です")
	}
	fmt.Printf("✅  count=%d\n", len(all))

	fmt.Print("[17] PeriodRepository::Remove... ")
	must("Remove", periodRepo.Remove(ctx, pid))
	gone, _ := periodRepo.FindByID(ctx, pid)
	if gone != nil {
		log.Fatal("❌ Remove: 削除後も期間が存在します")
	}
	fmt.Println("✅")

	must("team remove(post)", teamRepo.Remove(ctx, tid))
	must("user remove(post)", userRepo.Remove(ctx, uid))
}

// ─── [18-21] ObjectiveRepository ─────────────────────────────────────────────

func testObjectiveRepository(ctx context.Context, pool *pgxpool.Pool) {
	userRepo      := infrarepo.NewPgUserRepository(pool)
	teamRepo      := infrarepo.NewPgTeamRepository(pool)
	periodRepo    := infrarepo.NewPgPeriodRepository(pool)
	objectiveRepo := infrarepo.NewPgObjectiveRepository(pool)

	userUUID   := "aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa"
	teamUUID   := "bbbbbbbb-bbbb-4bbb-abbb-bbbbbbbbbbbb"
	memberUUID := "cccccccc-cccc-4ccc-accc-cccccccccccc"
	periodUUID := "dddddddd-dddd-4ddd-addd-dddddddddddd"
	objUUID    := "eeeeeeee-eeee-4eee-aeee-eeeeeeeeeeee"

	uid, _ := user.NewUserId(userUUID)
	tid, _ := team.NewTeamId(teamUUID)
	mid, _ := team.NewTeamMemberId(memberUUID)
	pid, _ := period.NewPeriodId(periodUUID)
	oid, _ := objective.NewObjectiveId(objUUID)

	// cleanup
	must("obj remove(pre)",    objectiveRepo.Remove(ctx, oid))
	must("period remove(pre)", periodRepo.Remove(ctx, pid))
	must("team remove(pre)",   teamRepo.Remove(ctx, tid))
	must("user remove(pre)",   userRepo.Remove(ctx, uid))

	// 前提データ
	email, _ := user.NewEmail("obj-go-smoke@example.com")
	u, _ := user.NewUser(uid, "目標テストユーザーGo", email, "hash", time.Time{}, time.Time{})
	must("user save", userRepo.Save(ctx, u))
	t, _ := team.NewTeam(tid, "目標テストチームGo", nil, nil, time.Time{}, time.Time{})
	m, _ := team.NewTeamMember(mid, tid, uid, team.Admin(), time.Now())
	must("AddMember", t.AddMember(m))
	must("team save", teamRepo.Save(ctx, t))
	h, _ := period.NewHalf("H2")
	dr, _ := period.NewDateRange("2025-07-01", "2025-12-31")
	p, _ := period.NewPeriod(pid, tid, "2025-H2-Go", h, dr, time.Time{}, time.Time{})
	must("period save", periodRepo.Save(ctx, p))

	fmt.Print("[18] ObjectiveRepository::Save (新規)... ")
	desc := "Goスモークテスト用目標"
	o, err := objective.NewObjective(oid, pid, uid, "Goスモーク目標", &desc, 1, time.Time{}, time.Time{})
	must("NewObjective", err)
	must("ObjectiveRepo.Save", objectiveRepo.Save(ctx, o))
	fmt.Println("✅")

	fmt.Print("[19] ObjectiveRepository::FindByID... ")
	found, err := objectiveRepo.FindByID(ctx, oid)
	must("FindByID", err)
	if found == nil || found.Title() != "Goスモーク目標" {
		log.Fatalf("❌ FindByID: title=%v", found)
	}
	fmt.Printf("✅  title=%s\n", found.Title())

	fmt.Print("[20] ObjectiveRepository::FindByPeriodID... ")
	byPeriod, err := objectiveRepo.FindByPeriodID(ctx, pid)
	must("FindByPeriodID", err)
	if len(byPeriod) == 0 {
		log.Fatal("❌ FindByPeriodID: 結果が0件です")
	}
	fmt.Printf("✅  count=%d\n", len(byPeriod))

	fmt.Print("[21] ObjectiveRepository::Remove... ")
	must("Remove", objectiveRepo.Remove(ctx, oid))
	gone, _ := objectiveRepo.FindByID(ctx, oid)
	if gone != nil {
		log.Fatal("❌ Remove: 削除後も目標が存在します")
	}
	fmt.Println("✅")

	must("period remove(post)", periodRepo.Remove(ctx, pid))
	must("team remove(post)",   teamRepo.Remove(ctx, tid))
	must("user remove(post)",   userRepo.Remove(ctx, uid))
}

// ─── [22-26] KeyResultRepository ─────────────────────────────────────────────

func testKeyResultRepository(ctx context.Context, pool *pgxpool.Pool) {
	userRepo      := infrarepo.NewPgUserRepository(pool)
	teamRepo      := infrarepo.NewPgTeamRepository(pool)
	periodRepo    := infrarepo.NewPgPeriodRepository(pool)
	objectiveRepo := infrarepo.NewPgObjectiveRepository(pool)
	krRepo        := infrarepo.NewPgKeyResultRepository(pool)

	userUUID   := "f1111111-1111-4111-a111-111111111111"
	teamUUID   := "f2222222-2222-4222-a222-222222222222"
	memberUUID := "f3333333-3333-4333-a333-333333333333"
	periodUUID := "f4444444-4444-4444-a444-444444444444"
	objUUID    := "f5555555-5555-4555-a555-555555555555"
	krUUID     := "f6666666-6666-4666-a666-666666666666"
	logUUID    := "f7777777-7777-4777-a777-777777777777"

	uid, _ := user.NewUserId(userUUID)
	tid, _ := team.NewTeamId(teamUUID)
	mid, _ := team.NewTeamMemberId(memberUUID)
	pid, _ := period.NewPeriodId(periodUUID)
	oid, _ := objective.NewObjectiveId(objUUID)
	kid, _ := keyresult.NewKeyResultId(krUUID)
	lid, _ := keyresult.NewKrProgressLogId(logUUID)

	// cleanup
	must("kr remove(pre)",     krRepo.Remove(ctx, kid))
	must("obj remove(pre)",    objectiveRepo.Remove(ctx, oid))
	must("period remove(pre)", periodRepo.Remove(ctx, pid))
	must("team remove(pre)",   teamRepo.Remove(ctx, tid))
	must("user remove(pre)",   userRepo.Remove(ctx, uid))

	// 前提データ
	email, _ := user.NewEmail("kr-go-smoke@example.com")
	u, _ := user.NewUser(uid, "KRテストユーザーGo", email, "hash", time.Time{}, time.Time{})
	must("user save", userRepo.Save(ctx, u))
	t, _ := team.NewTeam(tid, "KRテストチームGo", nil, nil, time.Time{}, time.Time{})
	m, _ := team.NewTeamMember(mid, tid, uid, team.Admin(), time.Now())
	must("AddMember", t.AddMember(m))
	must("team save", teamRepo.Save(ctx, t))
	h, _ := period.NewHalf("H1")
	dr, _ := period.NewDateRange("2026-01-01", "2026-06-30")
	p, _ := period.NewPeriod(pid, tid, "2026-H1-Go", h, dr, time.Time{}, time.Time{})
	must("period save", periodRepo.Save(ctx, p))
	o, _ := objective.NewObjective(oid, pid, uid, "KRテスト目標Go", nil, 0, time.Time{}, time.Time{})
	must("obj save", objectiveRepo.Save(ctx, o))

	// KR 生成（numeric）
	fmt.Print("[22] KeyResultRepository::Save (新規 numeric)... ")
	kr, err := keyresult.NewNumericKeyResult(kid, oid, uid, "売上 100万円", 100.0, 0, time.Time{}, time.Time{})
	must("NewNumericKeyResult", err)
	must("KrRepo.Save", krRepo.Save(ctx, kr))
	fmt.Println("✅")

	// FindByID
	fmt.Print("[23] KeyResultRepository::FindByID... ")
	found, err := krRepo.FindByID(ctx, kid)
	must("FindByID", err)
	if found == nil || found.Title() != "売上 100万円" {
		log.Fatalf("❌ FindByID: title=%v", found)
	}
	fmt.Printf("✅  title=%s\n", found.Title())

	// Save with progress log
	fmt.Print("[24] KeyResultRepository::Save (進捗ログ付き)... ")
	log_ := keyresult.NewNumericProgressLog(lid, kid, uid, 50.0, nil, time.Now())
	must("UpdateNumericProgress", kr.UpdateNumericProgress(log_))
	must("KrRepo.Save(with log)", krRepo.Save(ctx, kr))
	withLog, err := krRepo.FindByID(ctx, kid)
	must("FindByID(with log)", err)
	if len(withLog.ProgressLogs()) == 0 {
		log.Fatal("❌ 進捗ログが保存されていません")
	}
	fmt.Printf("✅  logs=%d\n", len(withLog.ProgressLogs()))

	// FindByObjectiveID
	fmt.Print("[25] KeyResultRepository::FindByObjectiveID... ")
	byObj, err := krRepo.FindByObjectiveID(ctx, oid)
	must("FindByObjectiveID", err)
	if len(byObj) == 0 {
		log.Fatal("❌ FindByObjectiveID: 結果が0件です")
	}
	fmt.Printf("✅  count=%d\n", len(byObj))

	// Remove
	fmt.Print("[26] KeyResultRepository::Remove... ")
	must("Remove", krRepo.Remove(ctx, kid))
	gone, _ := krRepo.FindByID(ctx, kid)
	if gone != nil {
		log.Fatal("❌ Remove: 削除後もKRが存在します")
	}
	fmt.Println("✅")

	must("obj remove(post)",    objectiveRepo.Remove(ctx, oid))
	must("period remove(post)", periodRepo.Remove(ctx, pid))
	must("team remove(post)",   teamRepo.Remove(ctx, tid))
	must("user remove(post)",   userRepo.Remove(ctx, uid))
}
