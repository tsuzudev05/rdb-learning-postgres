#!/bin/bash
# =============================================================================
# test.sh — C++ スモークテスト実行スクリプト
# DevContainer 内で実行する。
#
# 使い方:
#   bash /workspace/scripts/test.sh
# =============================================================================
set -e

WORKSPACE=/workspace
DDD_DIR=$WORKSPACE/05_DDD統合
BUILD_DIR=$DDD_DIR/build
TARGET=$BUILD_DIR/okr_smoke_test

echo "========================================"
echo " C++ スモークテスト"
echo "========================================"

# ─── DB 接続確認 ───
echo "[事前確認] PostgreSQL 接続確認..."
pg_isready -h postgres -U postgres -d learning \
  || { echo "❌ PostgreSQL に接続できません。DevContainer を再起動してください。"; exit 1; }
echo "✅ DB 接続 OK"
echo ""

cd "$DDD_DIR"

# ─── ビルド ───
echo "[1/2] C++ ビルド中..."
mkdir -p "$BUILD_DIR"
PQXX_CFLAGS=$(pkg-config --cflags libpqxx 2>/dev/null || echo "")
PQXX_LIBS=$(pkg-config --libs libpqxx 2>/dev/null || echo "-lpqxx -lpq")
g++ -std=c++17 -Wall -Wextra \
    $PQXX_CFLAGS \
    -I./src \
    src/main.cpp \
    $PQXX_LIBS \
    -o "$TARGET"
echo "✅ ビルド成功: $TARGET"
echo ""

# ─── テスト実行 ───
echo "[2/2] スモークテスト実行中..."
DATABASE_URL=postgresql://postgres:pass@postgres:5432/learning "$TARGET"

echo ""
echo "========================================"
echo " 完了"
echo "========================================"
