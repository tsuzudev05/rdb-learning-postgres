# =============================================================================
# Makefile — OKR Management Tool Go ビルド
# DevContainer 内の /workspace/05_DDD統合/go/ で実行する
#
# 使い方:
#   make init       DB初期化（Claude Code環境でコンテナ再起動後に実行）
#   make smoke      スモークテスト用バイナリをビルド
#   make run-smoke  スモークテスト実行（DB 接続必要）
#   make api        API サーバーバイナリをビルド（フェーズ8）
#   make run-api    API サーバーを起動（DB 接続必要）
#   make smoke-api  API エンドポイント疎通確認（curl）※ run-api と別ターミナルで実行
#   make build-all  smoke + api をまとめてビルド
# =============================================================================

DB_URL  := postgresql://postgres:pass@postgres:5432/learning
SCHEMA  := /workspace/05_DDD統合/schema.sql

.PHONY: init smoke run-smoke api run-api build-all smoke-api

# ── 初期化（Claude Code 環境：コンテナ再起動のたびに実行）────────────────────
# DevContainer（VS Code）では postgres コンテナが Docker Compose で自動起動するため不要。
init:
	@echo "🔧 PostgreSQL を起動します..."
	@pg_lsclusters | grep -q " online " || pg_ctlcluster 16 main start
	@grep -q "127.0.0.1 postgres" /etc/hosts || echo "127.0.0.1 postgres" >> /etc/hosts
	@echo "🔧 データベース・スキーマを初期化します..."
	@su -c "createdb learning 2>/dev/null || true" postgres
	@su -c "psql -d learning -f $(SCHEMA) 2>/dev/null || true" postgres
	@echo "✅ 初期化完了。make run-api でサーバーを起動できます。"

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

# ── API 疎通確認（curl ベース）────────────────────────────────────────────────
# API サーバーが起動している状態で実行する（別ターミナルで make run-api）
smoke-api:
	@echo "🔍 API エンドポイント疎通確認..."
	@bash scripts/smoke-api.sh

# ── まとめてビルド ─────────────────────────────────────────────────────────────
build-all: smoke api
