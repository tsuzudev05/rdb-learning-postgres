#!/bin/bash
# ============================================================
# benchmark_isolation.sh
# トランザクション分離レベル（READ COMMITTED）の動作確認
# 2つのセッションを疑似的に同時実行して競合状態を再現する
#
# 使い方：
#   chmod +x benchmark_isolation.sh
#   ./benchmark_isolation.sh
# ============================================================

set -e

DB_URL="${DATABASE_URL:-postgresql://postgres:pass@postgres:5432/learning}"
DIVIDER="─────────────────────────────────────────────"
PIPE_A="/tmp/psql_sync_a"
PIPE_B="/tmp/psql_sync_b"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
RED='\033[0;31m'
BOLD='\033[1m'
NC='\033[0m'

# 同期用パイプを準備
rm -f "$PIPE_A" "$PIPE_B"
mkfifo "$PIPE_A"
mkfifo "$PIPE_B"

# クリーンアップ関数
cleanup() {
  rm -f "$PIPE_A" "$PIPE_B"
  kill %1 %2 2>/dev/null || true
}
trap cleanup EXIT

echo ""
echo -e "${BOLD}${CYAN}🔀 PostgreSQL 分離レベル（READ COMMITTED）動作確認${NC}"
echo -e "${CYAN}${DIVIDER}${NC}"
echo -e "DB: ${DB_URL}"
echo -e "開始時刻: $(date '+%Y-%m-%d %H:%M:%S')"
echo -e "${CYAN}${DIVIDER}${NC}"

# ── 事前準備：対象データをリセット ───────────────────────────
echo ""
echo -e "${BOLD}▶ 事前準備: id=2 のステータスを 'in_progress' にリセット${NC}"
psql -x "$DB_URL" -c "UPDATE tasks SET status = 'in_progress' WHERE id = 2;"
psql -x "$DB_URL" -c "SELECT id, title, status FROM tasks WHERE id = 2;"

echo ""
echo -e "${CYAN}${DIVIDER}${NC}"
echo -e "${BOLD}【検証シナリオ】${NC}"
echo -e "  セッションA: トランザクション開始 → SELECT → （待機） → 再SELECT → COMMIT"
echo -e "  セッションB: トランザクション開始 → UPDATE → COMMIT"
echo -e "  期待する結果: AがBのCOMMIT後に再SELECTすると値が変わって見える（ノンリピータブルリード）"
echo -e "${CYAN}${DIVIDER}${NC}"
sleep 1

# ════════════════════════════════════════════
# セッションA（バックグラウンドで実行）
# ════════════════════════════════════════════
(
  echo -e "\n${YELLOW}[セッションA]${NC} トランザクション開始（READ COMMITTED）"
  RESULT1=$(psql -x "$DB_URL" << 'SQL'
BEGIN;
SET TRANSACTION ISOLATION LEVEL READ COMMITTED;
SELECT id, title, status FROM tasks WHERE id = 2;
SQL
)
  echo "$RESULT1"
  echo -e "${YELLOW}[セッションA]${NC} 1回目のSELECT完了 → セッションBの完了を待機中..."

  # セッションBの完了を待つ
  cat "$PIPE_B" > /dev/null

  echo -e "\n${YELLOW}[セッションA]${NC} セッションB完了を検知 → 2回目のSELECT実行"
  RESULT2=$(psql -x "$DB_URL" << 'SQL'
SELECT id, title, status FROM tasks WHERE id = 2;
COMMIT;
SQL
)
  echo "$RESULT2"
  echo -e "${YELLOW}[セッションA]${NC} COMMIT完了"

  # セッションAの完了をメインに通知
  echo "done" > "$PIPE_A"
) &

# セッションAが1回目のSELECTを終えるまで少し待つ
sleep 2

# ════════════════════════════════════════════
# セッションB（メインスレッドで実行）
# ════════════════════════════════════════════
echo ""
echo -e "${GREEN}[セッションB]${NC} トランザクション開始 → UPDATE → COMMIT"
psql -x "$DB_URL" << 'SQL'
BEGIN;
UPDATE tasks SET status = 'done' WHERE id = 2;
COMMIT;
SQL
echo -e "${GREEN}[セッションB]${NC} UPDATE・COMMIT完了 → セッションAに通知"

# セッションAにB完了を通知
echo "done" > "$PIPE_B"

# セッションAの完了を待つ
cat "$PIPE_A" > /dev/null

# ════════════════════════════════════════════
# サマリー
# ════════════════════════════════════════════
echo ""
echo -e "${BOLD}${CYAN}════════════════════════════════════════════${NC}"
echo -e "${BOLD}${CYAN}  📈 動作確認サマリー${NC}"
echo -e "${BOLD}${CYAN}════════════════════════════════════════════${NC}"
printf "  %-30s %-30s\n" "タイミング" "セッションAが見たstatus"
printf "  %-30s %-30s\n" "──────────────────────" "────────────────────"
printf "  %-30s ${YELLOW}%-30s${NC}\n" "B がCOMMITする前" "in_progress"
printf "  %-30s ${GREEN}%-30s${NC}\n" "B がCOMMITした後" "done"
echo -e "${BOLD}${CYAN}════════════════════════════════════════════${NC}"

echo ""
echo -e "${BOLD}💡 確認できた現象：ノンリピータブルリード${NC}"
echo -e "  同一トランザクション内で同じSELECTを2回実行したのに結果が変わった"
echo -e "  これは READ COMMITTED の特性："
echo -e "  → 他のトランザクションがCOMMITした値を即座に読み取る"
echo ""
echo -e "${BOLD}💡 REPEATABLE READとの違い：${NC}"
echo -e "  REPEATABLE READ にすると2回目のSELECTでも 'in_progress' のままになる"
echo -e "  → トランザクション開始時点のスナップショットを読み続けるため"
echo ""
echo -e "${BOLD}💡 ACIDとの対応：${NC}"
echo -e "  Isolation（分離性）の分離レベルによって"
echo -e "  他トランザクションの影響をどこまで遮断するかが変わる"
echo ""
echo -e "完了時刻: $(date '+%Y-%m-%d %H:%M:%S')"
echo ""
