#!/bin/bash
# ============================================================
# benchmark_transaction.sh
# トランザクション（BEGIN / ROLLBACK / COMMIT）の動作確認スクリプト
#
# 使い方：
#   chmod +x benchmark_transaction.sh
#   ./benchmark_transaction.sh
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

# -x オプション：列を縦に並べる拡張表示モード（横幅問題を回避）
PSQL="psql -x $DB_URL"

echo ""
echo -e "${BOLD}${CYAN}🔄 PostgreSQL トランザクション 動作確認${NC}"
echo -e "${CYAN}${DIVIDER}${NC}"
echo -e "DB: ${DB_URL}"
echo -e "開始時刻: $(date '+%Y-%m-%d %H:%M:%S')"
echo -e "${CYAN}${DIVIDER}${NC}"

# ── 事前確認：対象データ ──────────────────────────────────────
echo ""
echo -e "${BOLD}▶ 事前確認: 対象データ（tasks テーブル）${NC}"
echo -e "${CYAN}${DIVIDER}${NC}"
$PSQL -c "SELECT id, user_id, title, status FROM tasks ORDER BY id;"
echo -e "${CYAN}${DIVIDER}${NC}"

# ════════════════════════════════════════════
# ステップ1: ROLLBACK の確認
# ════════════════════════════════════════════
echo ""
echo -e "${BOLD}${YELLOW}════════════════════════════════════════════${NC}"
echo -e "${BOLD}${YELLOW}  ステップ1: ROLLBACK（なかったことにする）${NC}"
echo -e "${BOLD}${YELLOW}════════════════════════════════════════════${NC}"

echo ""
echo -e "${BOLD}▶ 1-1: トランザクション開始 → UPDATE → 変更を確認${NC}"
echo -e "${YELLOW}${DIVIDER}${NC}"
psql -x "$DB_URL" << 'SQL'
BEGIN;
UPDATE tasks SET status = 'done' WHERE user_id = 1;
SELECT id, title, status FROM tasks WHERE user_id = 1;
ROLLBACK;
SQL
echo -e "${YELLOW}${DIVIDER}${NC}"
echo -e "  ↑ BEGIN直後はstatusが 'done' に変わっている"

echo ""
echo -e "${BOLD}▶ 1-2: ROLLBACK後の確認（元に戻っているか）${NC}"
echo -e "${YELLOW}${DIVIDER}${NC}"
$PSQL -c "SELECT id, title, status FROM tasks WHERE user_id = 1;"
echo -e "${YELLOW}${DIVIDER}${NC}"
echo -e "  ${GREEN}✅ ROLLBACKにより元の状態に戻っている${NC}"

# ════════════════════════════════════════════
# ステップ2: COMMIT の確認
# ════════════════════════════════════════════
echo ""
echo -e "${BOLD}${GREEN}════════════════════════════════════════════${NC}"
echo -e "${BOLD}${GREEN}  ステップ2: COMMIT（変更を永続化する）${NC}"
echo -e "${BOLD}${GREEN}════════════════════════════════════════════${NC}"

echo ""
echo -e "${BOLD}▶ 2-1: トランザクション開始 → UPDATE → COMMIT${NC}"
echo -e "${GREEN}${DIVIDER}${NC}"
psql -x "$DB_URL" << 'SQL'
BEGIN;
UPDATE tasks SET status = 'in_progress' WHERE id = 2;
SELECT id, title, status FROM tasks WHERE id = 2;
COMMIT;
SQL
echo -e "${GREEN}${DIVIDER}${NC}"
echo -e "  ↑ COMMITでstatusが 'in_progress' に確定"

echo ""
echo -e "${BOLD}▶ 2-2: COMMIT後の確認（永続化されているか）${NC}"
echo -e "${GREEN}${DIVIDER}${NC}"
$PSQL -c "SELECT id, title, status FROM tasks WHERE id = 2;"
echo -e "${GREEN}${DIVIDER}${NC}"
echo -e "  ${GREEN}✅ COMMITにより変更が永続化されている${NC}"

# ── 最終確認：全データ ────────────────────────────────────────
echo ""
echo -e "${BOLD}▶ 最終確認: tasks テーブルの全データ${NC}"
echo -e "${CYAN}${DIVIDER}${NC}"
$PSQL -c "SELECT id, user_id, title, status FROM tasks ORDER BY id;"
echo -e "${CYAN}${DIVIDER}${NC}"

# ── サマリー ─────────────────────────────────────────────────
echo ""
echo -e "${BOLD}${CYAN}════════════════════════════════════════════${NC}"
echo -e "${BOLD}${CYAN}  📈 動作確認サマリー${NC}"
echo -e "${BOLD}${CYAN}════════════════════════════════════════════${NC}"
printf "  %-30s %-20s\n" "操作" "結果"
printf "  %-30s %-20s\n" "──────────────────────────" "──────────────"
printf "  %-30s ${RED}%-20s${NC}\n" "BEGIN → UPDATE → ROLLBACK" "変更が取り消された ✅"
printf "  %-30s ${GREEN}%-20s${NC}\n" "BEGIN → UPDATE → COMMIT" "変更が永続化された ✅"
echo -e "${BOLD}${CYAN}════════════════════════════════════════════${NC}"

echo ""
echo -e "${BOLD}💡 ACIDとの対応：${NC}"
echo -e "  Atomicity（原子性）: ROLLBACKで全部戻った → 中途半端な状態にならない"
echo -e "  Durability（永続性）: COMMITしたら再起動後も消えない"
echo ""
echo -e "完了時刻: $(date '+%Y-%m-%d %H:%M:%S')"
echo ""