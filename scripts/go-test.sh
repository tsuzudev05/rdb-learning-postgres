#!/bin/bash
# =============================================================================
# go-test.sh — Go ビルド・テストスクリプト
# DevContainer 内で実行する。
#
# 使い方:
#   bash /workspace/scripts/go-test.sh          # ビルド + テスト
#   bash /workspace/scripts/go-test.sh smoke    # ビルド + スモークテスト実行
# =============================================================================
set -e

GO_DIR="/workspace/05_DDD統合/go"

echo "========================================"
echo " Go ビルド・テスト"
echo "========================================"

if [ ! -d "$GO_DIR" ]; then
    echo "❌ Go モジュールが見つかりません: ${GO_DIR}"
    exit 1
fi

cd "$GO_DIR"
echo "📂 作業ディレクトリ: $(pwd)"
echo ""

# ─── ビルド ───
echo "[1/2] go build..."
go build ./...
echo "✅ ビルド成功"
echo ""

# ─── テスト ───
echo "[2/2] go test..."
go test ./...
echo "✅ テスト完了"

# ─── スモークテスト（DB 接続必要）───
if [ "${1}" = "smoke" ]; then
    echo ""
    echo "[オプション] スモークテスト実行中..."
    go run ./cmd/smoke
fi

echo ""
echo "========================================"
echo " 完了"
echo "========================================"
