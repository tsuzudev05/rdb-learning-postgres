// smoke — PgUserRepository の動作確認スクリプト
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

	// 1. 疎通確認
	testConnection(ctx, pool)

	// 2. UserRepository CRUD
	testUserRepository(ctx, pool)

	fmt.Println("========================================")
	fmt.Println(" ✅ 全テスト PASSED")
	fmt.Println("========================================")
}

// ─── test helpers ────────────────────────────────────────────────────────────

func must(label string, err error) {
	if err != nil {
		log.Fatalf("❌ %s: %v", label, err)
	}
}

func testConnection(ctx context.Context, pool *pgxpool.Pool) {
	fmt.Print("[1] DB 接続確認... ")
	var dbName string
	err := pool.QueryRow(ctx, "SELECT current_database()").Scan(&dbName)
	must("DB接続確認", err)
	fmt.Printf("✅  DB=%s\n", dbName)
}

func testUserRepository(ctx context.Context, pool *pgxpool.Pool) {
	repo := infrarepo.NewPgUserRepository(pool)

	const uuid = "22222222-2222-4222-a222-222222222222"
	id, err := user.NewUserId(uuid)
	must("NewUserId", err)
	email, err := user.NewEmail("gosmoke@example.com")
	must("NewEmail", err)

	// クリーンアップ（冪等）
	must("remove(pre-cleanup)", repo.Remove(ctx, id))

	// save（新規）
	fmt.Print("[2] Save (新規)... ")
	u, err := user.NewUser(id, "スモーク太郎Go", email, "hash_go_xxx", time.Time{}, time.Time{})
	must("NewUser", err)
	must("Save(new)", repo.Save(ctx, u))
	fmt.Println("✅")

	// FindByID
	fmt.Print("[3] FindByID... ")
	found, err := repo.FindByID(ctx, id)
	must("FindByID", err)
	if found == nil {
		log.Fatal("❌ FindByID: ユーザーが見つかりません")
	}
	if found.Name() != "スモーク太郎Go" {
		log.Fatalf("❌ FindByID: 名前が一致しません: %s", found.Name())
	}
	fmt.Printf("✅  name=%s\n", found.Name())

	// Save（upsert 更新）
	fmt.Print("[4] Save (更新)... ")
	updated, err := user.NewUser(id, "スモーク太郎Go更新済み", email, "hash_go_yyy", time.Time{}, time.Time{})
	must("NewUser(update)", err)
	must("Save(update)", repo.Save(ctx, updated))
	confirmed, err := repo.FindByID(ctx, id)
	must("FindByID(after update)", err)
	if confirmed == nil || confirmed.Name() != "スモーク太郎Go更新済み" {
		log.Fatalf("❌ Save(update): 更新が反映されていません")
	}
	fmt.Printf("✅  name=%s\n", confirmed.Name())

	// FindByEmail
	fmt.Print("[5] FindByEmail... ")
	byEmail, err := repo.FindByEmail(ctx, email)
	must("FindByEmail", err)
	if byEmail == nil || !byEmail.ID().Equal(id) {
		log.Fatal("❌ FindByEmail: IDが一致しません")
	}
	fmt.Println("✅")

	// FindAll
	fmt.Print("[6] FindAll... ")
	all, err := repo.FindAll(ctx)
	must("FindAll", err)
	if len(all) == 0 {
		log.Fatal("❌ FindAll: 結果が0件です")
	}
	fmt.Printf("✅  count=%d\n", len(all))

	// Remove
	fmt.Print("[7] Remove... ")
	must("Remove", repo.Remove(ctx, id))
	gone, err := repo.FindByID(ctx, id)
	must("FindByID(after remove)", err)
	if gone != nil {
		log.Fatal("❌ Remove: 削除後もユーザーが存在します")
	}
	fmt.Println("✅")
}
