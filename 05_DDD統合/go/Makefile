# =============================================================================
# Makefile — OKR Management Tool Go ビルド
# DevContainer 内の /workspace/05_DDD統合/go/ で実行する
#
# 使い方:
#   make smoke      スモークテスト用バイナリをビルド
#   make run-smoke  スモークテスト実行（DB 接続必要）
#   make api        API サーバーバイナリをビルド（フェーズ8）
#   make run-api    API サーバーを起動（DB 接続必要）
#   make build-all  smoke + api をまとめてビルド
# =============================================================================

DB_URL := postgresql://postgres:pass@postgres:5432/learning

.PHONY: smoke run-smoke api run-api build-all

# ── スモークテスト ─────────────────────────────────────────────────────────────
smoke:
	go build ./cmd/smoke/...
	@echo "✅ smoke ビルド成功"

run-smoke:
	@echo "🚀 スモークテスト実行..."
	DATABASE_URL=$(DB_URL) go run ./cmd/smoke

# ── API サーバー（フェーズ8）─────────────────────────────────────────────────
api:
	go build ./cmd/api/...
	@echo "✅ api ビルド成功"

run-api:
	@echo "🚀 OKR API サーバーを起動します: http://localhost:8080"
	DATABASE_URL=$(DB_URL) go run ./cmd/api

# ── まとめてビルド ─────────────────────────────────────────────────────────────
build-all: smoke api
