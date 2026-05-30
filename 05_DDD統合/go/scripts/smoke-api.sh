#!/usr/bin/env bash
# smoke-api.sh — OKR API エンドポイント疎通確認スクリプト
#
# 使い方（DevContainer 内で API サーバーが起動している状態で実行）:
#   bash scripts/smoke-api.sh
#   make smoke-api   # Makefile 経由
#
# API_BASE を変更すれば別ホスト・ポートにも向けられる。
set -euo pipefail

API_BASE="${API_BASE:-http://localhost:8080/api/v1}"
PASS=0
FAIL=0

# ── ヘルパー関数 ──────────────────────────────────────────────────────────────

check() {
  local label="$1"
  local expected="$2"
  local actual="$3"

  if echo "$actual" | grep -q "$expected"; then
    echo "  ✅ $label"
    PASS=$((PASS + 1))
  else
    echo "  ❌ $label"
    echo "     期待: $expected"
    echo "     実際: $(echo "$actual" | head -c 200)"
    FAIL=$((FAIL + 1))
  fi
}

post() {
  curl -s -X POST "$1" \
    -H "Content-Type: application/json" \
    -d "$2"
}

get()  { curl -s "$1"; }
del()  { curl -s -o /dev/null -w "%{http_code}" -X DELETE "$1"; }

echo ""
echo "🚀 OKR API 疎通確認 → $API_BASE"
echo "============================================="

# ── ヘルスチェック ─────────────────────────────────────────────────────────────
echo ""
echo "[ ヘルスチェック ]"
RES=$(get "$API_BASE/")
check "GET /api/v1/ → ok" '"ok"' "$RES"

# ── User ─────────────────────────────────────────────────────────────────────
echo ""
echo "[ User ]"
RES=$(post "$API_BASE/users" '{"name":"smoke-user","email":"smoke@example.com","password_hash":"hash"}')
check "POST /users → id あり" '"id"' "$RES"
USER_ID=$(echo "$RES" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

RES=$(get "$API_BASE/users")
check "GET /users → 配列" '\[' "$RES"

if [ -n "$USER_ID" ]; then
  RES=$(get "$API_BASE/users/$USER_ID")
  check "GET /users/:id → name" 'smoke-user' "$RES"

  CODE=$(del "$API_BASE/users/$USER_ID")
  check "DELETE /users/:id → 204" "204" "$CODE"
fi

# ── Team ─────────────────────────────────────────────────────────────────────
echo ""
echo "[ Team ]"
RES=$(post "$API_BASE/teams" '{"name":"smoke-team"}')
check "POST /teams → id あり" '"id"' "$RES"
TEAM_ID=$(echo "$RES" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

RES=$(get "$API_BASE/teams")
check "GET /teams → 配列" '\[' "$RES"

if [ -n "$TEAM_ID" ]; then
  RES=$(get "$API_BASE/teams/$TEAM_ID")
  check "GET /teams/:id → name" 'smoke-team' "$RES"

  CODE=$(del "$API_BASE/teams/$TEAM_ID")
  check "DELETE /teams/:id → 204" "204" "$CODE"
fi

# ── Period ────────────────────────────────────────────────────────────────────
echo ""
echo "[ Period ]"
# Period には team_id が必要なため、テスト用チームを再作成する
RES=$(post "$API_BASE/teams" '{"name":"smoke-team-for-period"}')
TEAM_ID=$(echo "$RES" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

if [ -n "$TEAM_ID" ]; then
  PERIOD_BODY=$(printf '{"team_id":"%s","name":"2026上期","half":"H1","start_date":"2026-04-01","end_date":"2026-09-30"}' "$TEAM_ID")
  RES=$(post "$API_BASE/periods" "$PERIOD_BODY")
  check "POST /periods → id あり" '"id"' "$RES"
  PERIOD_ID=$(echo "$RES" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

  RES=$(get "$API_BASE/periods?team_id=$TEAM_ID")
  check "GET /periods?team_id → 配列" '\[' "$RES"

  if [ -n "$PERIOD_ID" ]; then
    RES=$(get "$API_BASE/periods/$PERIOD_ID")
    check "GET /periods/:id → name" '2026上期' "$RES"

    CODE=$(del "$API_BASE/periods/$PERIOD_ID")
    check "DELETE /periods/:id → 204" "204" "$CODE"
  fi

  del "$API_BASE/teams/$TEAM_ID" > /dev/null
fi

# ── Objective ─────────────────────────────────────────────────────────────────
echo ""
echo "[ Objective ]"
# Objective には period_id + owner_id が必要なため再作成する
RES_USER=$(post "$API_BASE/users" '{"name":"obj-owner","email":"objowner@example.com","password_hash":"h"}')
OBJ_USER_ID=$(echo "$RES_USER" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

RES_TEAM=$(post "$API_BASE/teams" '{"name":"obj-team"}')
OBJ_TEAM_ID=$(echo "$RES_TEAM" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

if [ -n "$OBJ_TEAM_ID" ]; then
  PERIOD_BODY=$(printf '{"team_id":"%s","name":"obj-period","half":"H2","start_date":"2026-10-01","end_date":"2027-03-31"}' "$OBJ_TEAM_ID")
  RES_PERIOD=$(post "$API_BASE/periods" "$PERIOD_BODY")
  OBJ_PERIOD_ID=$(echo "$RES_PERIOD" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

  if [ -n "$OBJ_PERIOD_ID" ] && [ -n "$OBJ_USER_ID" ]; then
    OBJ_BODY=$(printf '{"period_id":"%s","owner_id":"%s","title":"smoke-objective","display_order":0}' "$OBJ_PERIOD_ID" "$OBJ_USER_ID")
    RES=$(post "$API_BASE/objectives" "$OBJ_BODY")
    check "POST /objectives → id あり" '"id"' "$RES"
    OBJ_ID=$(echo "$RES" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

    RES=$(get "$API_BASE/objectives?period_id=$OBJ_PERIOD_ID")
    check "GET /objectives?period_id → 配列" '\[' "$RES"

    RES=$(get "$API_BASE/objectives?owner_id=$OBJ_USER_ID")
    check "GET /objectives?owner_id → 配列" '\[' "$RES"

    if [ -n "$OBJ_ID" ]; then
      RES=$(get "$API_BASE/objectives/$OBJ_ID")
      check "GET /objectives/:id → title" 'smoke-objective' "$RES"

      CODE=$(del "$API_BASE/objectives/$OBJ_ID")
      check "DELETE /objectives/:id → 204" "204" "$CODE"
    fi

    del "$API_BASE/periods/$OBJ_PERIOD_ID" > /dev/null
  fi

  del "$API_BASE/teams/$OBJ_TEAM_ID" > /dev/null
fi
[ -n "$OBJ_USER_ID" ] && del "$API_BASE/users/$OBJ_USER_ID" > /dev/null || true

# ── KeyResult ─────────────────────────────────────────────────────────────────
echo ""
echo "[ KeyResult ]"
echo "  ⏭  TODO: フェーズ8-4 実装後に追加"

# ── 結果サマリー ───────────────────────────────────────────────────────────────
echo ""
echo "============================================="
echo "結果: ✅ $PASS 件成功 / ❌ $FAIL 件失敗"
echo ""
if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
