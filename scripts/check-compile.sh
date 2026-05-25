#!/bin/bash
# =============================================================================
# check-compile.sh — C++ ヘッダーのコンパイル確認（DB 接続不要）
#
# main.cpp に #include されていないヘッダー（ユースケース層など）を
# スタブ .cpp 経由でコンパイルチェックする。
#
# DevContainer 内で実行する。
#
# 使い方:
#   bash /workspace/scripts/check-compile.sh
# =============================================================================
set -e

DDD_DIR="/workspace/05_DDD統合"
SRC="${DDD_DIR}/src"
BUILD_DIR="${DDD_DIR}/build"
STUB="${BUILD_DIR}/_compile_check_stub.cpp"

mkdir -p "$BUILD_DIR"

# libpqxx のコンパイルフラグ（ユースケース層は pqxx 不要だが infrastructure 層との
# 依存解決のため一応取得しておく）
PQXX_CFLAGS=$(pkg-config --cflags libpqxx 2>/dev/null || echo "")
PQXX_LIBS=$(pkg-config --libs   libpqxx 2>/dev/null || echo "-lpqxx -lpq")

echo "========================================"
echo " C++ ヘッダーコンパイル確認"
echo " （DB 接続不要）"
echo "========================================"
echo ""

PASS=0
FAIL=0

# ヘッダーを1つコンパイルチェックする関数
# 引数: <ヘッダーの相対パス（SRCからの相対）> <表示ラベル>
check_header() {
    local header="$1"
    local label="$2"
    echo -n "  [check] ${label} ... "

    # スタブ .cpp を生成してコンパイルし、バイナリは捨てる
    printf '#include "%s"\nint main(){return 0;}\n' "$header" > "$STUB"

    if g++ -std=c++17 -Wall -Wextra \
           $PQXX_CFLAGS \
           -I"$SRC" \
           "$STUB" \
           $PQXX_LIBS \
           -o /dev/null 2>&1; then
        echo "✅"
        PASS=$((PASS + 1))
    else
        echo "❌ コンパイルエラー"
        FAIL=$((FAIL + 1))
    fi

    rm -f "$STUB"
}

# ─── チェック対象ヘッダー一覧 ───────────────────────────────────────────────
# フェーズ6: ユースケース層
check_header "application/usecase/UserUseCase.hpp"   "フェーズ6 UserUseCase"

# フェーズ4: インフラ層（スモークテストで間接的にビルドされるが念のため）
check_header "infrastructure/repository/PgUserRepository.hpp"   "フェーズ4 PgUserRepository"
check_header "infrastructure/repository/PgTeamRepository.hpp"   "フェーズ4 PgTeamRepository"
check_header "infrastructure/repository/PgPeriodRepository.hpp" "フェーズ4 PgPeriodRepository"

# ─── 結果サマリー ─────────────────────────────────────────────────────────────
echo ""
echo "========================================"
if [ "$FAIL" -eq 0 ]; then
    echo " ✅ 全 $((PASS)) 件 コンパイル確認 PASSED"
else
    echo " ❌ $FAIL 件 失敗 / $((PASS + FAIL)) 件中"
fi
echo "========================================"

exit $FAIL
