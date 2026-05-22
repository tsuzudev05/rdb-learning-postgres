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

echo "========================================"
echo " C++ スモークテスト"
echo "========================================"

# ─── DB 接続確認 ───
echo "[事前確認] PostgreSQL 接続確認..."
pg_isready -h postgres -U postgres -d learning \
  || { echo "❌ PostgreSQL に接続できません。DevContainer を再起動してください。"; exit 1; }
echo "✅ DB 接続 OK"
echo ""

# ─── ビルド ───
echo "[1/2] C++ ビルド中..."
cd "$DDD_DIR"
make build
echo ""

# ─── テスト実行 ───
echo "[2/2] スモークテスト実行中..."
make test

echo ""
echo "========================================"
echo " 完了"
echo "========================================"
