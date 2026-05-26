#!/bin/bash
# =============================================================================
# run-cli.sh — OKR管理 CLI の起動スクリプト（フェーズ7）
#
# ビルド済みバイナリがなければ自動でビルドしてから起動する。
# DevContainer 内で実行する。
#
# 使い方:
#   bash /workspace/scripts/run-cli.sh
#
# 環境変数:
#   DATABASE_URL  接続文字列（未設定時は DevContainer デフォルト値を使用）
# =============================================================================
set -e

SCRIPTS_DIR="$(cd "$(dirname "$0")" && pwd)"
WORKSPACE="/workspace/05_DDD統合"
OUTPUT="${WORKSPACE}/build/okr-cli"
DB_URL="${DATABASE_URL:-postgresql://postgres:pass@postgres:5432/learning}"

# ─── DB 接続確認 ─────────────────────────────────────────────────────────────
echo "🔍 PostgreSQL 接続確認..."
pg_isready -h postgres -U postgres -d learning \
  || { echo "❌ PostgreSQL に接続できません。DevContainer を再起動してください。"; exit 1; }
echo "✅ DB 接続 OK"
echo ""

# ─── ビルド（バイナリが存在しない場合のみ）──────────────────────────────────
if [ ! -f "${OUTPUT}" ]; then
    echo "🔨 バイナリが見つかりません。ビルドします..."
    bash "${SCRIPTS_DIR}/build-cli.sh"
    echo ""
fi

# ─── CLI 起動 ────────────────────────────────────────────────────────────────
echo "🖥️  OKR CLI 起動"
echo "   DB : ${DB_URL}"
echo "   'help' でコマンド一覧  'quit' で終了"
echo ""

DATABASE_URL="${DB_URL}" "${OUTPUT}"
