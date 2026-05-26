# =============================================================================
# Makefile — ローカルDocker操作ショートカット
#
# 使い方（プロジェクトルートで実行）:
#   make up       コンテナ起動（初回はイメージビルドも実行）
#   make down     コンテナ停止・削除
#   make shell    app コンテナに bash で入る
#   make build    コンテナ内で C++ をビルド
#   make test     コンテナ内でスモークテストを実行
#   make db       コンテナ内で psql を起動
#   make db-check コンテナ内で DB 接続確認
#   make logs     コンテナのログを tail
#   make rebuild  イメージを再ビルドしてコンテナを起動
# =============================================================================

COMPOSE := docker compose
APP     := $(COMPOSE) exec app

.PHONY: up down shell build test cli run-cli db db-check logs rebuild setup

up:
	$(COMPOSE) up -d
	@echo ""
	@echo "✅ コンテナ起動完了"
	@echo "   make shell   → app コンテナに入る"
	@echo "   make build   → C++ ビルド"
	@echo "   make test    → スモークテスト実行"

down:
	$(COMPOSE) down

rebuild:
	$(COMPOSE) down
	$(COMPOSE) build --no-cache
	$(COMPOSE) up -d

shell:
	$(APP) bash

setup:
	$(APP) bash /workspace/scripts/setup.sh

build:
	$(APP) bash -c "cd /workspace/05_DDD統合 && make build"

test:
	$(APP) bash -c "cd /workspace/05_DDD統合 && make run"

cli:
	$(APP) bash -c "cd /workspace/05_DDD統合 && make cli"

run-cli:
	$(APP) bash -c "cd /workspace/05_DDD統合 && make run-cli"

db:
	$(APP) psql postgresql://postgres:pass@postgres:5432/learning

db-check:
	$(APP) bash /workspace/scripts/db-check.sh

logs:
	$(COMPOSE) logs -f
