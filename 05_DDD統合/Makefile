# =============================================================================
# Makefile — OKR Management Tool C++ ビルド
# DevContainer 内の /workspace/05_DDD統合/ で実行する
#
# 使い方:
#   make          ビルド（デフォルト: スモークテスト用バイナリ）
#   make build    スモークテスト用バイナリをビルド
#   make test     ビルド + スモークテスト実行
#   make cli      CLI バイナリをビルド（フェーズ7）
#   make run-cli  CLI バイナリをビルドして起動
#   make clean    ビルド成果物を削除
# =============================================================================

CXX      := g++
CXXFLAGS := -std=c++17 -Wall -Wextra
INCLUDES := -I./src $(shell pkg-config --cflags libpqxx 2>/dev/null)
LIBS     := $(shell pkg-config --libs libpqxx 2>/dev/null || echo "-lpqxx -lpq")

BUILD_DIR   := ./build
TARGET      := $(BUILD_DIR)/okr_smoke_test
TARGET_CLI  := $(BUILD_DIR)/okr-cli

DB_URL := postgresql://postgres:pass@postgres:5432/learning

.PHONY: all build test cli run-cli clean

all: build

# ── スモークテスト用バイナリ ─────────────────────────────────────────────────
build: $(TARGET)

$(TARGET): src/main.cpp
	mkdir -p $(BUILD_DIR)
	$(CXX) $(CXXFLAGS) $(INCLUDES) $< $(LIBS) -o $@
	@echo "✅ ビルド成功: $@"

test: build
	@echo "🚀 スモークテスト実行..."
	DATABASE_URL=$(DB_URL) $(TARGET)

# ── CLI バイナリ（フェーズ7）────────────────────────────────────────────────
cli: $(TARGET_CLI)

$(TARGET_CLI): src/main_cli.cpp
	mkdir -p $(BUILD_DIR)
	$(CXX) $(CXXFLAGS) $(INCLUDES) $< $(LIBS) -o $@
	@echo "✅ CLI ビルド成功: $@"

run-cli: cli
	@echo "🖥️  OKR CLI 起動..."
	DATABASE_URL=$(DB_URL) $(TARGET_CLI)

# ── クリーン ─────────────────────────────────────────────────────────────────
clean:
	rm -rf $(BUILD_DIR)
	@echo "🧹 ビルドディレクトリを削除しました"
