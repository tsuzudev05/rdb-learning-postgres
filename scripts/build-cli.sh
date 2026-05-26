#!/bin/bash
# =============================================================================
# build-cli.sh — CLI バイナリのビルドスクリプト（フェーズ7）
#
# DevContainer 内で実行する。
#
# 使い方:
#   bash /workspace/scripts/build-cli.sh          # ビルドのみ
#   bash /workspace/scripts/build-cli.sh run      # ビルド + CLI 起動
# =============================================================================
set -e

WORKSPACE="/workspace/05_DDD統合"
SRC="${WORKSPACE}/src"
BUILD="${WORKSPACE}/build"
OUTPUT="${BUILD}/okr-cli"

mkdir -p "$BUILD"

echo "🔨 CLI ビルド開始..."
echo "   src     : ${SRC}/main_cli.cpp"
echo "   output  : ${OUTPUT}"

PQXX_CFLAGS=$(pkg-config --cflags libpqxx 2>/dev/null || echo "-I/usr/include")
PQXX_LIBS=$(pkg-config --libs libpqxx 2>/dev/null || echo "-lpqxx -lpq")

g++ -std=c++17 -Wall -Wextra \
    ${PQXX_CFLAGS} \
    -I"${SRC}" \
    "${SRC}/main_cli.cpp" \
    ${PQXX_LIBS} \
    -o "${OUTPUT}"

echo "✅ CLI ビルド成功: ${OUTPUT}"

if [ "${1}" = "run" ]; then
    echo ""
    echo "🖥️  OKR CLI 起動..."
    echo "   'help' でコマンド一覧  'quit' で終了"
    echo ""
    DATABASE_URL="${DATABASE_URL:-postgresql://postgres:pass@postgres:5432/learning}" \
        "${OUTPUT}"
fi
