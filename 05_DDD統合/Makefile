# =============================================================================
# Makefile — OKR Management Tool C++ ビルド
# DevContainer 内の /workspace/05_DDD統合/ で実行する
#
# 使い方:
#   make          ビルド（デフォルト）
#   make build    ビルド
#   make test     ビルド + スモークテスト実行
#   make clean    ビルド成果物を削除
# =============================================================================

CXX      := g++
CXXFLAGS := -std=c++17 -Wall -Wextra
INCLUDES := -I./src $(shell pkg-config --cflags libpqxx 2>/dev/null)
LIBS     := $(shell pkg-config --libs libpqxx 2>/dev/null || echo "-lpqxx -lpq")

BUILD_DIR := ./build
TARGET    := $(BUILD_DIR)/okr_smoke_test

.PHONY: all build test clean

all: build

build: $(TARGET)

$(TARGET): src/main.cpp
	mkdir -p $(BUILD_DIR)
	$(CXX) $(CXXFLAGS) $(INCLUDES) $< $(LIBS) -o $@
	@echo "✅ ビルド成功: $@"

test: build
	@echo "🚀 スモークテスト実行..."
	DATABASE_URL=postgresql://postgres:pass@postgres:5432/learning $(TARGET)

clean:
	rm -rf $(BUILD_DIR)
	@echo "🧹 ビルドディレクトリを削除しました"
