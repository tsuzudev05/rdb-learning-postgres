#!/bin/bash
# =============================================================================
# cowork-test.sh — Cowork サンドボックスでの C++ スモークテスト
#
# root・Docker・sudo 不要で完結する。
# apt-get download で libpq / libpqxx / PostgreSQL 14 を /tmp に展開して実行。
#
# 使い方:
#   bash /workspace/scripts/cowork-test.sh
# =============================================================================
set -e

LOCAL=/tmp/local
PG_BIN=$LOCAL/usr/lib/postgresql/14/bin
PG_DATA=/tmp/pgdata
export LD_LIBRARY_PATH=$LOCAL/usr/lib/x86_64-linux-gnu:$LOCAL/usr/lib/postgresql/14/lib:/usr/lib/x86_64-linux-gnu

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SRCDIR="$REPO_ROOT/05_DDD統合/src"
SCHEMA="$REPO_ROOT/05_DDD統合/schema.sql"
BUILD=/tmp/build

echo "========================================"
echo " Cowork スモークテスト"
echo "========================================"

# ─── 依存ライブラリのセットアップ（未インストール時のみ）───
if [ ! -f "$PG_BIN/postgres" ]; then
  echo "[setup] 依存パッケージを取得中（初回のみ）..."
  mkdir -p $LOCAL
  cd /tmp
  apt-get download libpq5 libpq-dev libpqxx-dev libpqxx-6.4 \
                   postgresql-14 postgresql-client-14 2>/dev/null
  for deb in libpq5_*.deb libpq-dev_*.deb libpqxx*.deb postgresql-14_*.deb postgresql-client-14_*.deb; do
    [ -f "$deb" ] && dpkg-deb -x "$deb" $LOCAL
  done
  echo "✅ 依存パッケージ展開完了"
fi

# ─── PostgreSQL 起動 ───
if [ ! -d "$PG_DATA" ]; then
  echo "[1/5] DB クラスター初期化..."
  $PG_BIN/initdb -D $PG_DATA -U postgres --auth=trust --no-locale --encoding=UTF8 -q
fi

echo "[1/5] PostgreSQL 起動..."
$PG_BIN/pg_ctl -D $PG_DATA -l /tmp/pg.log \
  -o "-p 5555 -k /tmp -c listen_addresses=''" start -w -t 15

# ─── DB + スキーマ準備 ───
echo "[2/5] DB・スキーマ準備..."
PSQL="$PG_BIN/psql -h /tmp -p 5555 -U postgres"
$PSQL -c "CREATE DATABASE learning;" 2>/dev/null || true
$PSQL -d learning -f "$SCHEMA" > /dev/null 2>&1
echo "✅ スキーマ投入完了"

# ─── C++ コンパイル ───
echo "[3/5] C++ コンパイル..."
mkdir -p $BUILD
g++ -std=c++17 -Wall \
    -I$LOCAL/usr/include \
    -I$LOCAL/usr/include/postgresql \
    -I$SRCDIR \
    "$SRCDIR/main.cpp" \
    -L$LOCAL/usr/lib/x86_64-linux-gnu \
    -lpqxx -lpq \
    -Wl,-rpath,$LOCAL/usr/lib/x86_64-linux-gnu \
    -o $BUILD/okr_smoke_test
echo "✅ ビルド成功"

# ─── スモークテスト実行 ───
echo "[4/5] テスト実行..."
DATABASE_URL="postgresql://postgres@/learning?host=/tmp&port=5555" \
  $BUILD/okr_smoke_test

# ─── PostgreSQL 停止 ───
echo "[5/5] PostgreSQL 停止..."
$PG_BIN/pg_ctl -D $PG_DATA stop -m fast > /dev/null 2>&1
echo "✅ 完了"
