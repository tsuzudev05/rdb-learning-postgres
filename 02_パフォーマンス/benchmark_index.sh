#!/bin/bash
# ============================================================
# benchmark_index.sh
# インデックス有無によるクエリ実行時間の比較スクリプト
#
# 使い方：
#   chmod +x benchmark_index.sh
#   ./benchmark_index.sh
# ============================================================

set -e

DB_URL="${DATABASE_URL:-postgresql://postgres:pass@postgres:5432/learning}"
DIVIDER="─────────────────────────────────────────────"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
RED='\033[0;31m'
BOLD='\033[1m'
NC='\033[0m'

echo ""
echo -e "${BOLD}${CYAN}📊 PostgreSQL インデックス ベンチマーク${NC}"
echo -e "${CYAN}${DIVIDER}${NC}"
echo -e "DB: ${DB_URL}"
echo -e "開始時刻: $(date '+%Y-%m-%d %H:%M:%S')"
echo -e "${CYAN}${DIVIDER}${NC}"

# ── Step 1: データ準備 ──────────────────────────────────────
echo ""
echo -e "${BOLD}▶ Step 1: データ準備${NC}"

BEFORE=$(psql "$DB_URL" -t -c "SELECT COUNT(*) FROM users;" | tr -d ' \n')
echo -e "  現在のレコード数: ${BEFORE}件"

if [ "$BEFORE" -lt 10000 ]; then
  echo -e "  10,000件のデータを投入中..."
  START=$(date +%s%3N)
  psql "$DB_URL" -q << 'SQL'
INSERT INTO users (name, email)
SELECT '田中' || i, 'user' || i || '@example.com'
FROM generate_series(1, 10000) AS i
ON CONFLICT DO NOTHING;
SQL
  END=$(date +%s%3N)
  echo -e "  ${GREEN}✅ 投入完了（所要時間: $((END - START))ms）${NC}"
else
  echo -e "  ${GREEN}✅ データ投入済み（スキップ）${NC}"
fi

AFTER=$(psql "$DB_URL" -t -c "SELECT COUNT(*) FROM users;" | tr -d ' \n')
echo -e "  総レコード数: ${AFTER}件"

# ── Step 2: インデックス削除 ──────────────────────────────────
echo ""
echo -e "${BOLD}▶ Step 2: インデックスを削除してクリーンな状態に戻す${NC}"
psql "$DB_URL" -q -c "DROP INDEX IF EXISTS idx_users_name;" 2>/dev/null
echo -e "  ${GREEN}✅ idx_users_name を削除（なければスキップ）${NC}"

# ── Step 3: インデックスなしで計測 ────────────────────────────
echo ""
echo -e "${BOLD}▶ Step 3: インデックスなし（Seq Scan）${NC}"
echo -e "${YELLOW}${DIVIDER}${NC}"

RESULT_NO_INDEX=$(psql "$DB_URL" -c "EXPLAIN ANALYZE SELECT * FROM users WHERE name = '田中5000';")
echo "$RESULT_NO_INDEX"

EXEC_TIME_NO_INDEX=$(echo "$RESULT_NO_INDEX" | grep "Execution Time" | grep -oP '[0-9]+\.[0-9]+')
SCAN_TYPE_NO_INDEX=$(echo "$RESULT_NO_INDEX" | grep -oP 'Seq Scan|Index Scan|Bitmap' | head -1)
# Fix: "Rows Removed by Filter: 10002" から数字だけ抽出
ROWS_REMOVED=$(echo "$RESULT_NO_INDEX" | grep "Rows Removed by Filter" | grep -oP '[0-9]+')

echo -e "${YELLOW}${DIVIDER}${NC}"
echo -e "  スキャン方式: ${RED}${SCAN_TYPE_NO_INDEX}${NC}"
echo -e "  除去した行数: ${ROWS_REMOVED}件"
echo -e "  実行時間:     ${RED}${EXEC_TIME_NO_INDEX} ms${NC}"

# ── Step 4: インデックス作成 ──────────────────────────────────
echo ""
echo -e "${BOLD}▶ Step 4: インデックスを作成${NC}"
START=$(date +%s%3N)
psql "$DB_URL" -q -c "CREATE INDEX idx_users_name ON users(name);"
END=$(date +%s%3N)
echo -e "  ${GREEN}✅ idx_users_name 作成完了（所要時間: $((END - START))ms）${NC}"

# ── Step 5: インデックスありで計測 ────────────────────────────
echo ""
echo -e "${BOLD}▶ Step 5: インデックスあり（Index Scan）${NC}"
echo -e "${GREEN}${DIVIDER}${NC}"

RESULT_WITH_INDEX=$(psql "$DB_URL" -c "EXPLAIN ANALYZE SELECT * FROM users WHERE name = '田中5000';")
echo "$RESULT_WITH_INDEX"

EXEC_TIME_WITH_INDEX=$(echo "$RESULT_WITH_INDEX" | grep "Execution Time" | grep -oP '[0-9]+\.[0-9]+')
SCAN_TYPE_WITH_INDEX=$(echo "$RESULT_WITH_INDEX" | grep -oP 'Seq Scan|Index Scan|Bitmap' | head -1)

echo -e "${GREEN}${DIVIDER}${NC}"
echo -e "  スキャン方式: ${GREEN}${SCAN_TYPE_WITH_INDEX}${NC}"
echo -e "  実行時間:     ${GREEN}${EXEC_TIME_WITH_INDEX} ms${NC}"

# ── Step 6: 結果サマリー ──────────────────────────────────────
echo ""
echo -e "${BOLD}${CYAN}════════════════════════════════════════════${NC}"
echo -e "${BOLD}${CYAN}  📈 ベンチマーク結果サマリー${NC}"
echo -e "${BOLD}${CYAN}════════════════════════════════════════════${NC}"
printf "  %-20s %-20s %-20s\n" "項目" "インデックスなし" "インデックスあり"
printf "  %-20s %-20s %-20s\n" "────────────────" "────────────────" "────────────────"
printf "  %-20s %-20s %-20s\n" "スキャン方式" "${SCAN_TYPE_NO_INDEX}" "${SCAN_TYPE_WITH_INDEX}"
printf "  %-20s %-20s %-20s\n" "実行時間" "${EXEC_TIME_NO_INDEX} ms" "${EXEC_TIME_WITH_INDEX} ms"
echo -e "${BOLD}${CYAN}════════════════════════════════════════════${NC}"

# Fix: bcの代わりにawkで高速化倍率を計算
IMPROVEMENT=$(awk "BEGIN {printf \"%.1f\", $EXEC_TIME_NO_INDEX / $EXEC_TIME_WITH_INDEX}")
echo -e "  🚀 高速化倍率: ${BOLD}${GREEN}約 ${IMPROVEMENT} 倍${NC}"

echo ""
echo -e "完了時刻: $(date '+%Y-%m-%d %H:%M:%S')"
echo ""