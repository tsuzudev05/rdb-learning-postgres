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

if ! git config --global user.name &>/dev/null; then
  git config --global user.name "${GIT_USER_NAME}"
fi
if ! git config --global user.email &>/dev/null; then
  git config --global user.email "${GIT_USER_EMAIL}"
fi

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
# 3. DDD スキーマの投入（05_DDD統合/schema.sql を使用）
#    users テーブルが存在しない場合のみ投入する（再起動時のデータ消失を防ぐ）
# ─────────────────────────────────────────
echo "📊 DDD スキーマを確認中..."
TABLE_EXISTS=$(psql postgresql://postgres:pass@postgres:5432/learning -tAc "SELECT to_regclass('public.users')")
if [ -z "$TABLE_EXISTS" ]; then
  echo "  テーブルが存在しないためスキーマを投入します..."
  psql postgresql://postgres:pass@postgres:5432/learning \
    -f /workspace/05_DDD統合/schema.sql \
  && echo "✅ DDD スキーマ投入完了" \
  || echo "  ⚠️  スキーマ投入でエラーが発生しました"
else
  echo "✅ DDD スキーマ投入済み（スキップ）"
fi

# ─────────────────────────────────────────
# 4. C++ ビルドディレクトリの準備
# ─────────────────────────────────────────
echo "🔨 C++ ビルドディレクトリを準備中..."
mkdir -p /workspace/05_DDD統合/build
chmod +x /workspace/scripts/*.sh 2>/dev/null || true
echo "✅ 準備完了"

# ─────────────────────────────────────────
# 5. バージョン確認
# ─────────────────────────────────────────
echo ""
echo "📌 インストール済みツール："
echo "  g++     : $(g++ --version | head -1)"
echo "  make    : $(make --version | head -1)"
echo "  libpqxx : $(pkg-config --modversion libpqxx 2>/dev/null || echo '確認中...')"
echo "  Go      : $(go version)"
echo "  Rust    : $(rustc --version)"
echo "  Node.js : $(node --version)"
echo "  psql    : $(psql --version)"
echo "  git     : $(git --version)"
echo ""
echo "✅ セットアップ完了！"
echo ""
echo "💡 テストを実行するには："
echo "   $ bash /workspace/scripts/test.sh"
echo ""
echo "🤖 Claude Code を使うには："
echo "   $ claude          # 対話モードで起動"
echo "   $ claude login    # 初回ログイン（ブラウザ認証）"
