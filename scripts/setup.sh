#!/bin/bash
# コンテナ初回セットアップスクリプト
# 使い方: make setup  または  bash /workspace/scripts/setup.sh
set -e

echo "🚀 開発環境セットアップ開始..."

# ─────────────────────────────────────────
# 1. Git 設定
# ─────────────────────────────────────────
echo "🔧 Git設定中..."
git config --global safe.directory /workspace

if [ -d /root/.ssh ]; then
  chmod 700 /root/.ssh
  find /root/.ssh -type f -name "*.pub" -exec chmod 644 {} \;
  find /root/.ssh -type f ! -name "*.pub" -exec chmod 600 {} \; 2>/dev/null || true
fi

if ! git config --global user.name &>/dev/null; then
  git config --global user.name "tsuzudev05"
fi
if ! git config --global user.email &>/dev/null; then
  git config --global user.email "tsuzu.develop.05@gmail.com"
fi
if ! git config --global credential.helper &>/dev/null; then
  git config --global credential.helper store
fi
echo "✅ Git設定完了"

# ─────────────────────────────────────────
# 2. スクリプトに実行権限を付与
# ─────────────────────────────────────────
chmod +x /workspace/scripts/*.sh 2>/dev/null || true
echo "✅ scripts/ 実行権限付与完了"

# ─────────────────────────────────────────
# 3. PostgreSQL 接続確認
# ─────────────────────────────────────────
echo "📦 PostgreSQL への接続を確認中..."
until pg_isready -h "${POSTGRES_HOST:-postgres}" -U postgres -d learning; do
  echo "  待機中..."
  sleep 2
done
echo "✅ PostgreSQL に接続できました"

# ─────────────────────────────────────────
# 4. C++ ビルドディレクトリの準備
# ─────────────────────────────────────────
echo "🔨 C++ ビルドディレクトリを準備中..."
mkdir -p /workspace/05_DDD統合/build
echo "✅ build/ ディレクトリ作成完了"

# ─────────────────────────────────────────
# 5. バージョン確認
# ─────────────────────────────────────────
echo ""
echo "📌 インストール済みツール："
echo "  g++     : $(g++ --version | head -1)"
echo "  libpqxx : $(pkg-config --modversion libpqxx 2>/dev/null || echo '確認中...')"
echo "  Go      : $(go version)"
echo "  Rust    : $(rustc --version)"
echo "  Node.js : $(node --version)"
echo "  psql    : $(psql --version)"
echo "  git     : $(git --version)"
echo ""
echo "✅ セットアップ完了！"
echo ""
echo "💡 次のステップ："
echo "   make build  → C++ ビルド（プロジェクトルートから）"
echo "   make test   → スモークテスト実行"
echo "   make db     → psql で直接DB接続"
