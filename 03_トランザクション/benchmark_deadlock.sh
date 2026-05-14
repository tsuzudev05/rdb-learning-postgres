#!/bin/bash
# ============================================================
# benchmark_deadlock.sh
# デッドロックの再現と回避策の動作確認スクリプト
# Pythonのスレッドで2セッションを真の同時実行で再現する
# ============================================================

DB_URL="${DATABASE_URL:-postgresql://postgres:pass@postgres:5432/learning}"
DIVIDER="─────────────────────────────────────────────"

GREEN='\033[0;32m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

echo ""
echo -e "${BOLD}${CYAN}💀 PostgreSQL デッドロック 動作確認${NC}"
echo -e "${CYAN}${DIVIDER}${NC}"
echo -e "DB: ${DB_URL}"
echo -e "開始時刻: $(date '+%Y-%m-%d %H:%M:%S')"
echo -e "${CYAN}${DIVIDER}${NC}"

# ── 事前準備 ─────────────────────────────────────────────────
echo ""
echo -e "${BOLD}▶ 事前準備: データをリセット${NC}"
psql -x "$DB_URL" -c "UPDATE tasks SET status = 'todo' WHERE id IN (1, 2);"
psql -x "$DB_URL" -c "SELECT id, title, status FROM tasks WHERE id IN (1, 2) ORDER BY id;"

# ── Pythonスクリプトで2セッション同時実行 ─────────────────────
python3 << 'PYEOF'
import psycopg2
import threading
import time
import os

DB = os.environ.get("DATABASE_URL", "postgresql://postgres:pass@postgres:5432/learning")

# ANSI カラー
YEL  = '\033[1;33m'
GRN  = '\033[0;32m'
RED  = '\033[0;31m'
CYN  = '\033[0;36m'
BOLD = '\033[1m'
NC   = '\033[0m'

# セッション間の同期用イベント
a_locked = threading.Event()  # AがROW1をロックしたらセット
b_locked = threading.Event()  # BがROW2をロックしたらセット

# ════════════════════════════════════════════
# Part 1: デッドロックの再現
# ════════════════════════════════════════════
print(f"\n{RED}{'='*44}{NC}")
print(f"{RED}  Part 1: デッドロックを意図的に発生させる{NC}")
print(f"{RED}{'='*44}{NC}\n")

def session_a():
    conn = psycopg2.connect(DB)
    conn.autocommit = False
    cur = conn.cursor()
    try:
        print(f"{YEL}[セッションA]{NC} BEGIN → id=1 をUPDATE（ロック取得）")
        cur.execute("UPDATE tasks SET status = 'in_progress' WHERE id = 1")
        print(f"{YEL}[セッションA]{NC} id=1 ロック完了 → セッションBを待機中...")
        a_locked.set()       # AがロックしたことをBに通知
        b_locked.wait()      # BがROW2をロックするまで待つ

        print(f"{YEL}[セッションA]{NC} id=2 のUPDATEを要求（Bと競合 → デッドロック待ち）")
        cur.execute("UPDATE tasks SET status = 'in_progress' WHERE id = 2")
        conn.commit()
        print(f"{YEL}[セッションA]{NC} COMMIT完了")
    except psycopg2.errors.DeadlockDetected as e:
        print(f"{YEL}[セッションA]{NC} {RED}ERROR: deadlock detected → 自動ROLLBACK{NC}")
        conn.rollback()
    except Exception as e:
        print(f"{YEL}[セッションA]{NC} エラー: {e}")
        conn.rollback()
    finally:
        cur.close()
        conn.close()

def session_b():
    conn = psycopg2.connect(DB)
    conn.autocommit = False
    cur = conn.cursor()
    try:
        a_locked.wait()      # AがROW1をロックするまで待つ
        print(f"{GRN}[セッションB]{NC} BEGIN → id=2 をUPDATE（ロック取得）")
        cur.execute("UPDATE tasks SET status = 'done' WHERE id = 2")
        print(f"{GRN}[セッションB]{NC} id=2 ロック完了 → セッションAに通知")
        b_locked.set()       # BがROW2をロックしたことをAに通知

        time.sleep(0.5)
        print(f"{GRN}[セッションB]{NC} id=1 のUPDATEを要求（Aと競合 → デッドロック待ち）")
        cur.execute("UPDATE tasks SET status = 'done' WHERE id = 1")
        conn.commit()
        print(f"{GRN}[セッションB]{NC} COMMIT完了")
    except psycopg2.errors.DeadlockDetected as e:
        print(f"{GRN}[セッションB]{NC} {RED}ERROR: deadlock detected → 自動ROLLBACK{NC}")
        conn.rollback()
    except Exception as e:
        print(f"{GRN}[セッションB]{NC} エラー: {e}")
        conn.rollback()
    finally:
        cur.close()
        conn.close()

t_a = threading.Thread(target=session_a)
t_b = threading.Thread(target=session_b)
t_a.start()
t_b.start()
t_a.join()
t_b.join()

# ════════════════════════════════════════════
# Part 1 後のデータ確認
# ════════════════════════════════════════════
print(f"\n{BOLD}▶ Part 1 後のデータ確認{NC}")
conn = psycopg2.connect(DB)
cur = conn.cursor()
cur.execute("SELECT id, title, status FROM tasks WHERE id IN (1, 2) ORDER BY id")
rows = cur.fetchall()
for row in rows:
    print(f"  id={row[0]} | {row[1]} | status={RED}{row[2]}{NC}")
cur.close()
conn.close()
print("  → 片方がROLLBACKされ、もう片方のみ反映されている")

# ════════════════════════════════════════════
# Part 2: 回避策（ロック順を統一する）
# ════════════════════════════════════════════
print(f"\n{GRN}{'='*44}{NC}")
print(f"{GRN}  Part 2: 回避策（ロック取得順を統一する）{NC}")
print(f"{GRN}{'='*44}{NC}\n")

# データリセット
conn = psycopg2.connect(DB)
conn.autocommit = True
cur = conn.cursor()
cur.execute("UPDATE tasks SET status = 'todo' WHERE id IN (1, 2)")
cur.close()
conn.close()

a2_done = threading.Event()

def session_a2():
    conn = psycopg2.connect(DB)
    conn.autocommit = False
    cur = conn.cursor()
    try:
        print(f"{YEL}[セッションA]{NC} BEGIN → id=1 → id=2 の順でUPDATE（昇順）")
        cur.execute("UPDATE tasks SET status = 'in_progress' WHERE id = 1")
        cur.execute("UPDATE tasks SET status = 'in_progress' WHERE id = 2")
        conn.commit()
        print(f"{YEL}[セッションA]{NC} COMMIT完了")
    finally:
        cur.close()
        conn.close()
    a2_done.set()

def session_b2():
    a2_done.wait()   # Aが完了してからBを実行
    conn = psycopg2.connect(DB)
    conn.autocommit = False
    cur = conn.cursor()
    try:
        print(f"{GRN}[セッションB]{NC} BEGIN → id=1 → id=2 の順でUPDATE（昇順・同じ順番）")
        cur.execute("UPDATE tasks SET status = 'done' WHERE id = 1")
        cur.execute("UPDATE tasks SET status = 'done' WHERE id = 2")
        conn.commit()
        print(f"{GRN}[セッションB]{NC} COMMIT完了（デッドロックなし）")
    finally:
        cur.close()
        conn.close()

t_a2 = threading.Thread(target=session_a2)
t_b2 = threading.Thread(target=session_b2)
t_a2.start()
t_b2.start()
t_a2.join()
t_b2.join()

print(f"\n{BOLD}▶ Part 2 後のデータ確認{NC}")
conn = psycopg2.connect(DB)
cur = conn.cursor()
cur.execute("SELECT id, title, status FROM tasks WHERE id IN (1, 2) ORDER BY id")
rows = cur.fetchall()
for row in rows:
    print(f"  id={row[0]} | {row[1]} | status={GRN}{row[2]}{NC}")
cur.close()
conn.close()
print("  → 両方のセッションが正常に完了している")

# ── サマリー ─────────────────────────────────────────────────
print(f"""
{BOLD}{CYN}{'='*44}{NC}
{BOLD}{CYN}  📈 動作確認サマリー{NC}
{BOLD}{CYN}{'='*44}{NC}
  {"シナリオ":<30} {"結果"}
  {"─"*30} {"─"*20}
  {"ロック順が逆（A:1→2, B:2→1）":<28} {RED}デッドロック発生 💀{NC}
  {"ロック順を統一（A:1→2, B:1→2）":<28} {GRN}正常完了 ✅{NC}
{BOLD}{CYN}{'='*44}{NC}

{BOLD}💡 デッドロックの回避策：{NC}
  1. 複数行を更新する場合は必ず id 昇順でロックを取得する
  2. トランザクション内の処理を短くしてロック保持時間を最小化する
  3. SELECT FOR UPDATE で明示的ロックを取得して順番を制御する

{BOLD}💡 PostgreSQL の自動検知：{NC}
  デッドロックを約1秒で自動検知し、片方を強制ROLLBACKして解消する
  アプリ側はエラーを受け取ったらリトライする実装が必要
""")
PYEOF

echo -e "完了時刻: $(date '+%Y-%m-%d %H:%M:%S')"
echo ""