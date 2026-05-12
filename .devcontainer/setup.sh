#!/bin/bash
set -e

echo "🚀 開発環境セットアップ開始..."

# ─────────────────────────────────────────
# 1. PostgreSQL 接続確認
# ─────────────────────────────────────────
echo "📦 PostgreSQL への接続を確認中..."
until pg_isready -h postgres -U postgres -d learning; do
  echo "  待機中..."
  sleep 2
done
echo "✅ PostgreSQL に接続できました"

# ─────────────────────────────────────────
# 2. 初期スキーマの投入
# ─────────────────────────────────────────
echo "📊 初期スキーマを投入中..."
psql postgresql://postgres:pass@postgres:5432/learning \
  -f /workspace/.devcontainer/init/01_init_schema.sql \
  2>/dev/null || echo "  スキーマ投入済み（スキップ）"

# ─────────────────────────────────────────
# 3. バージョン確認
# ─────────────────────────────────────────
echo ""
echo "📌 インストール済みツール："
echo "  Node.js : $(node --version)"
echo "  claude  : $(claude --version 2>/dev/null || echo '確認中...')"
echo "  Go      : $(go version)"
echo "  Rust    : $(rustc --version)"
echo "  psql    : $(psql --version)"
echo ""
echo "✅ セットアップ完了！"
echo ""
echo "🤖 Claude Code を使うには："
echo "   $ claude          # 対話モードで起動"
echo "   $ claude login    # 初回ログイン（ブラウザ認証）"
