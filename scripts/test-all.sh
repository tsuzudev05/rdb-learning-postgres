#!/bin/bash
# =============================================================================
# test-all.sh — C++ + Go 両方をビルド・テストするスクリプト
# DevContainer 内で実行する。
#
# 使い方:
#   bash /workspace/scripts/test-all.sh
# =============================================================================
set -e

SCRIPTS_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "========================================"
echo " 全体テスト (C++ + Go)"
echo "========================================"
echo ""

# ─── DB 接続確認 ───
echo "[事前確認] PostgreSQL 接続確認..."
pg_isready -h postgres -U postgres -d learning \
  || { echo "❌ PostgreSQL に接続できません。DevContainer を再起動してください。"; exit 1; }
echo "✅ DB 接続 OK"
echo ""

# ─── C++ ───
echo "--- C++ ---"
bash "${SCRIPTS_DIR}/test.sh"
echo ""

# ─── Go ───
echo "--- Go ---"
bash "${SCRIPTS_DIR}/go-test.sh"

echo ""
echo "========================================"
echo " 全体テスト 完了"
echo "========================================"
