// api — OKR Management API サーバー（echo v4）
//
// DevContainer 内での実行:
//
//	cd /workspace/05_DDD統合/go
//	go get github.com/labstack/echo/v4   # 初回のみ（go.mod / go.sum 更新）
//	make run-api
//	# または
//	DATABASE_URL=postgresql://postgres:pass@postgres:5432/learning go run ./cmd/api
//
// 環境変数 DATABASE_URL が設定されていない場合は既定の接続文字列を使用する。
package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/tsuzudev05/rdb-learning-postgres/okr/handler"
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

	// ── DB 接続 ──────────────────────────────────────────────────────────────────
	pool, err := pgxpool.New(ctx, connString())
	if err != nil {
		log.Fatalf("DB接続失敗: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("DB疎通確認失敗: %v", err)
	}
	log.Println("✅ DB 接続成功")

	// ── Repository（インフラ層）──────────────────────────────────────────────────
	// フェーズ8-2 以降で各ハンドラーに渡す
	userRepo := infrarepo.NewPgUserRepository(pool)
	teamRepo := infrarepo.NewPgTeamRepository(pool)
	periodRepo := infrarepo.NewPgPeriodRepository(pool)
	objectiveRepo := infrarepo.NewPgObjectiveRepository(pool)
	keyResultRepo := infrarepo.NewPgKeyResultRepository(pool)

	// ── echo インスタンス・ミドルウェア設定 ─────────────────────────────────────
	e := echo.New()
	e.HideBanner = true

	e.Use(middleware.Logger())  // リクエストログ
	e.Use(middleware.Recover()) // パニックリカバリ

	// ── 統一エラーハンドリング（フェーズ8-4）────────────────────────────────────
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		code := http.StatusInternalServerError
		msg := "内部サーバーエラー"
		if he, ok := err.(*echo.HTTPError); ok {
			code = he.Code
			if s, ok := he.Message.(string); ok {
				msg = s
			}
		}
		if !c.Response().Committed {
			_ = c.JSON(code, map[string]interface{}{
				"error": map[string]interface{}{
					"code":    code,
					"message": msg,
				},
			})
		}
	}

	// ── ハンドラー（ユースケース層の代わりに Repository を直接 DI）──────────────
	userHandler        := handler.NewUserHandler(userRepo)
	teamHandler        := handler.NewTeamHandler(teamRepo)
	periodHandler      := handler.NewPeriodHandler(periodRepo)
	objectiveHandler   := handler.NewObjectiveHandler(objectiveRepo)
	keyResultHandler   := handler.NewKeyResultHandler(keyResultRepo)

	// ── ルーティング ─────────────────────────────────────────────────────────────
	v1 := e.Group("/api/v1")

	// ヘルスチェック
	v1.GET("/", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status":  "ok",
			"service": "okr-api",
			"version": "v1",
		})
	})

	// User エンドポイント（フェーズ8-2）
	v1.GET("/users", userHandler.List)
	v1.POST("/users", userHandler.Create)
	v1.GET("/users/:id", userHandler.Get)
	v1.DELETE("/users/:id", userHandler.Delete)

	// Team エンドポイント（フェーズ8-2）
	v1.GET("/teams", teamHandler.List)
	v1.POST("/teams", teamHandler.Create)
	v1.GET("/teams/:id", teamHandler.Get)
	v1.DELETE("/teams/:id", teamHandler.Delete)
	v1.POST("/teams/:id/members", teamHandler.AddMember)
	v1.DELETE("/teams/:id/members/:user_id", teamHandler.RemoveMember)

	// Period エンドポイント（フェーズ8-3）
	v1.GET("/periods", periodHandler.List)
	v1.POST("/periods", periodHandler.Create)
	v1.GET("/periods/:id", periodHandler.Get)
	v1.DELETE("/periods/:id", periodHandler.Delete)

	// Objective エンドポイント（フェーズ8-3）
	v1.GET("/objectives", objectiveHandler.List)
	v1.POST("/objectives", objectiveHandler.Create)
	v1.GET("/objectives/:id", objectiveHandler.Get)
	v1.DELETE("/objectives/:id", objectiveHandler.Delete)

	// KeyResult エンドポイント（フェーズ8-4）
	v1.GET("/key_results", keyResultHandler.List)
	v1.POST("/key_results", keyResultHandler.Create)
	v1.GET("/key_results/:id", keyResultHandler.Get)
	v1.DELETE("/key_results/:id", keyResultHandler.Delete)

	// ── サーバー起動 ─────────────────────────────────────────────────────────────
	addr := ":8080"
	log.Printf("🚀 OKR API サーバーを起動します: http://localhost%s", addr)
	if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
		log.Fatalf("サーバーエラー: %v", err)
	}
}
