#!/bin/bash
# C++ ビルドスクリプト
# 使い方:
#   ./scripts/build.sh          # ビルドのみ
#   ./scripts/build.sh run      # ビルド + 実行

set -e

WORKSPACE="/workspace/05_DDD統合"
SRC="${WORKSPACE}/src"
BUILD="${WORKSPACE}/build"
OUTPUT="${BUILD}/okr_smoke_test"

# ビルドディレクトリ作成
mkdir -p "$BUILD"

echo "🔨 C++ ビルド開始..."
echo "   src     : ${SRC}"
echo "   output  : ${OUTPUT}"

# libpqxx のコンパイルフラグを pkg-config で取得
PQXX_CFLAGS=$(pkg-config --cflags libpqxx 2>/dev/null || echo "-I/usr/include")
PQXX_LIBS=$(pkg-config --libs libpqxx 2>/dev/null || echo "-lpqxx -lpq")

g++ -std=c++17 -Wall -Wextra \
    ${PQXX_CFLAGS} \
    -I"${SRC}" \
    "${SRC}/main.cpp" \
    ${PQXX_LIBS} \
    -o "${OUTPUT}"

echo "✅ ビルド成功: ${OUTPUT}"

# 実行オプション
if [ "${1}" = "run" ]; then
    echo ""
    echo "🚀 スモークテスト実行中..."
    "${OUTPUT}"
fi
