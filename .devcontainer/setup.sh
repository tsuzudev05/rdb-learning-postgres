#!/bin/bash
set -e

echo "🚀 開発環境セットアップ開始..."

# ─────────────────────────────────────────
# 1. Git 設定
# ─────────────────────────────────────────
echo "🔧 Git設定中..."
git config --global safe.directory /workspace

# SSH キーのパーミッション修正（マウントされた場合）
if [ -d /root/.ssh ]; then
  chmod 700 /root/.ssh
  find /root/.ssh -type f -name "*.pub" -exec chmod 644 {} \;
  find /root/.ssh -type f ! -name "*.pub" -exec chmod 600 {} \; 2>/dev/null || true
fi

# グローバルgit設定にuser.name/emailが未設定なら固定値を設定する
if ! git config --global user.name &>/dev/null; then
  git config --global user.name "tsuzudev05"
fi
if ! git config --global user.email &>/dev/null; then
  git config --global user.email "tsuzu.develop.05@gmail.com"
fi

# credential.helper のフォールバック（VS Code 外から使う場合）
if ! git config --global credential.helper &>/dev/null; then
  git config --global credential.helper store
fi

echo "✅ Git設定完了"

# ─────────────────────────────────────────
# 2. PostgreSQL 接続確認
# ─────────────────────────────────────────
echo "📦 PostgreSQL への接続を確認中..."
until pg_isready -h postgres -U postgres -d learning; do
  echo "  待機中..."
  sleep 2
done
echo "✅ PostgreSQL に接続できました"

# ─────────────────────────────────────────
# 3. 初期スキーマの投入
# ─────────────────────────────────────────
echo "📊 初期スキーマを投入中..."
psql postgresql://postgres:pass@postgres:5432/learning \
  -f /workspace/.devcontainer/init/01_init_schema.sql \
  2>/dev/null || echo "  スキーマ投入済み（スキップ）"

# ─────────────────────────────────────────
# 4. バージョン確認
# ─────────────────────────────────────────
echo ""
echo "📌 インストール済みツール："
echo "  Node.js : $(node --version)"
echo "  claude  : $(claude --version 2>/dev/null || echo '確認中...')"
echo "  Go      : $(go version)"
echo "  Rust    : $(rustc --version)"
echo "  psql    : $(psql --version)"
echo "  git     : $(git --version)"
echo ""
echo "✅ セットアップ完了！"
echo ""
echo "🤖 Claude Code を使うには："
echo "   $ claude          # 対話モードで起動"
echo "   $ claude login    # 初回ログイン（ブラウザ認証）"
