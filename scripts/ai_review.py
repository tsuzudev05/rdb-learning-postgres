#!/usr/bin/env python3
"""
ai_review.py — PR diff を Groq（Llama 3.3 70B）に渡してコードレビューを生成し、
               GitHub PR にコメントとして投稿するスクリプト。

Groq 無料枠: 14,400 req/日・30 RPM（クレジットカード不要）

Usage (GitHub Actions から呼ばれる):
    python scripts/ai_review.py

必要な環境変数:
    GROQ_API_KEY   Groq API キー（GitHub Secret に登録）
    GITHUB_TOKEN   GitHub Actions が自動で提供
    PR_NUMBER      PR 番号
    REPO           "owner/repo" 形式
    PR_TITLE       PR タイトル（プロンプトのコンテキスト用）
    BASE_BRANCH    マージ先ブランチ名
    HEAD_BRANCH    PR のブランチ名
"""

import os
import re
import sys
import json
import urllib.request
import urllib.error
from datetime import datetime, timezone

from groq import Groq

# ─── 設定 ────────────────────────────────────────────────────────────────────

DIFF_FILE      = "diff.txt"
USAGE_FILE     = ".monthly-usage.json"
MAX_DIFF_CHARS = 80_000   # llama-3.3-70b-versatile の context 128K tokens を考慮
MODEL          = "llama-3.3-70b-versatile"

# Groq 無料枠は $0。有料プランの参考単価（2025年時点）
PRICE_INPUT_PER_MTOK  = 0.59   # USD / million tokens
PRICE_OUTPUT_PER_MTOK = 0.79   # USD / million tokens

MONTHLY_BUDGET_USD = 1.00   # 有料転換時の安全網（無料枠内なら実際には課金されない）
BUDGET_WARN_RATIO  = 0.80


# ─── コスト計算 ──────────────────────────────────────────────────────────────

def calc_cost(input_tokens: int, output_tokens: int) -> float:
    """トークン数から USD コストを計算する（Groq 有料プラン料金。無料枠内は $0）。"""
    return (
        input_tokens  / 1_000_000 * PRICE_INPUT_PER_MTOK
        + output_tokens / 1_000_000 * PRICE_OUTPUT_PER_MTOK
    )


# ─── 月次使用量の永続化 ──────────────────────────────────────────────────────

def current_month() -> str:
    return datetime.now(timezone.utc).strftime("%Y-%m")


def load_monthly_usage() -> dict:
    default = {"month": current_month(), "input_tokens": 0, "output_tokens": 0, "cost_usd": 0.0}
    if not os.path.exists(USAGE_FILE):
        return default
    try:
        with open(USAGE_FILE, encoding="utf-8") as f:
            data = json.load(f)
        if data.get("month") != current_month():
            print(f"[info] 月が変わりました（{data.get('month')} → {current_month()}）。使用量をリセットします。")
            return default
        return data
    except (json.JSONDecodeError, KeyError):
        return default


def save_monthly_usage(usage: dict) -> None:
    with open(USAGE_FILE, "w", encoding="utf-8") as f:
        json.dump(usage, f, ensure_ascii=False, indent=2)
    print(f"[info] 月次使用量を保存しました: ${usage['cost_usd']:.4f} / ${MONTHLY_BUDGET_USD:.2f}")


def check_budget(usage: dict) -> tuple[bool, str]:
    current_cost = usage["cost_usd"]
    remaining    = MONTHLY_BUDGET_USD - current_cost
    ratio        = current_cost / MONTHLY_BUDGET_USD

    if ratio >= 1.0:
        msg = (
            f"⚠️ **月次コスト上限に達しました**（${current_cost:.4f} / ${MONTHLY_BUDGET_USD:.2f}）。"
            f"AIレビューをスキップしました。来月リセットされます。"
        )
        return False, msg

    warn = ""
    if ratio >= BUDGET_WARN_RATIO:
        warn = (
            f"\n\n> ⚠️ 今月の使用コスト: **${current_cost:.4f}** / ${MONTHLY_BUDGET_USD:.2f} "
            f"（残り ${remaining:.4f}）"
        )
    return True, warn


# ─── プロンプト ───────────────────────────────────────────────────────────────

SYSTEM_PROMPT = """あなたは DDD（ドメイン駆動設計）・C++17・Go・PostgreSQL に詳しいシニアエンジニアです。
Pull Request の diff をレビューし、以下の観点で日本語でフィードバックしてください。

レビュー観点:
- コードの正確性・バグの可能性
- DDD / クリーンアーキテクチャの原則への準拠（集約・値オブジェクト・Repository の分離など）
- C++17 / Go のイディオムへの準拠
- セキュリティリスク（SQL インジェクション・認証情報の露出など）
- パフォーマンス上の懸念
- テストの考慮事項
- 改善提案（必須ではないが有益なもの）

出力フォーマット（Markdown）:
## 🤖 AI コードレビュー（Llama 3.3 70B / Groq）

### 概要
（PR で何をしているかを 2〜3 行で要約）

### ✅ 良い点
（箇条書き）

### 🔴 要修正
（深刻な問題があれば記載。なければ「なし」）

### 🟡 改善提案
（任意だが推奨する変更点）

### 📊 総評
（一言でマージ可否の判断：LGTM / 要修正 / 要確認）

---
*このレビューは Llama 3.3 70B（Groq）によって自動生成されました。*
"""


# ─── diff 読み込み ────────────────────────────────────────────────────────────

def load_diff() -> str:
    if not os.path.exists(DIFF_FILE):
        print(f"[error] {DIFF_FILE} が見つかりません", file=sys.stderr)
        sys.exit(1)

    with open(DIFF_FILE, encoding="utf-8", errors="replace") as f:
        diff = f.read()

    if not diff.strip():
        print("[info] diff が空です（変更なし）。スキップします。")
        sys.exit(0)

    if len(diff) > MAX_DIFF_CHARS:
        print(f"[warn] diff が {len(diff)} 文字で上限 {MAX_DIFF_CHARS} を超えています。切り詰めます。")
        diff = diff[:MAX_DIFF_CHARS] + "\n\n... (diff が長いため省略されました)"

    return diff


# ─── Groq API 呼び出し ────────────────────────────────────────────────────────

def generate_review(diff: str, pr_title: str, base: str, head: str) -> tuple[str, int, int]:
    """
    Groq（Llama 3.3 70B）を呼び出してレビューを生成する。
    戻り値: (レビュー本文, input_tokens, output_tokens)
    """
    client = Groq(api_key=sanitize_api_key(os.environ["GROQ_API_KEY"]))

    user_message = f"""以下の Pull Request をレビューしてください。

**PR タイトル**: {pr_title}
**ブランチ**: `{head}` → `{base}`

```diff
{diff}
```
"""

    print(f"[info] {MODEL}（Groq）にリクエスト中...")
    chat_completion = client.chat.completions.create(
        messages=[
            {"role": "system", "content": SYSTEM_PROMPT},
            {"role": "user", "content": user_message},
        ],
        model=MODEL,
        max_tokens=2048,
    )

    review        = chat_completion.choices[0].message.content
    input_tokens  = chat_completion.usage.prompt_tokens
    output_tokens = chat_completion.usage.completion_tokens
    cost          = calc_cost(input_tokens, output_tokens)

    print(f"[info] 完了 — input: {input_tokens} tok / output: {output_tokens} tok / 参考コスト: ${cost:.4f}")
    return review, input_tokens, output_tokens


# ─── GitHub コメント投稿 ──────────────────────────────────────────────────────

def post_github_comment(repo: str, pr_number: str, body: str) -> None:
    token   = os.environ["GITHUB_TOKEN"]
    url     = f"https://api.github.com/repos/{repo}/issues/{pr_number}/comments"
    data    = json.dumps({"body": body}).encode("utf-8")
    headers = {
        "Authorization": f"Bearer {token}",
        "Accept": "application/vnd.github+json",
        "Content-Type": "application/json",
        "X-GitHub-Api-Version": "2022-11-28",
    }

    req = urllib.request.Request(url, data=data, headers=headers, method="POST")
    try:
        with urllib.request.urlopen(req) as resp:
            result = json.loads(resp.read())
            print(f"[info] コメント投稿完了: {result['html_url']}")
    except urllib.error.HTTPError as e:
        print(f"[error] GitHub API エラー: {e.code} {e.read().decode()}", file=sys.stderr)
        sys.exit(1)


# ─── ユーティリティ ───────────────────────────────────────────────────────────

def sanitize_api_key(raw: str) -> str:
    """API キーから制御文字・BOM・改行などを除去する。"""
    cleaned = re.sub(r"[^\x21-\x7E]", "", raw)
    if not cleaned:
        print("[error] GROQ_API_KEY が空か、印字可能文字を含みません", file=sys.stderr)
        sys.exit(1)
    if len(raw) != len(cleaned):
        print(f"[info] API キーを sanitize しました（元: {len(raw)} 文字 → 後: {len(cleaned)} 文字）")
    return cleaned


# ─── メイン ──────────────────────────────────────────────────────────────────

def main() -> None:
    required_env = ["GROQ_API_KEY", "GITHUB_TOKEN", "PR_NUMBER", "REPO"]
    missing = [k for k in required_env if not os.environ.get(k)]
    if missing:
        print(f"[error] 環境変数が不足しています: {missing}", file=sys.stderr)
        sys.exit(1)

    pr_title  = os.environ.get("PR_TITLE", "（タイトルなし）")
    base      = os.environ.get("BASE_BRANCH", "main")
    head      = os.environ.get("HEAD_BRANCH", "feature")
    pr_number = os.environ["PR_NUMBER"]
    repo      = os.environ["REPO"]

    # 1. 月次コスト確認（無料枠内は実質 $0 だが安全網として残す）
    usage = load_monthly_usage()
    can_run, budget_msg = check_budget(usage)

    if not can_run:
        print(f"[warn] 予算超過のためスキップします: {budget_msg}")
        post_github_comment(repo, pr_number, budget_msg)
        sys.exit(0)

    # 2. diff を読み込む
    diff = load_diff()
    print(f"[info] diff: {len(diff)} 文字")

    # 3. Groq でレビュー生成
    review, input_tokens, output_tokens = generate_review(diff, pr_title, base, head)

    # 4. 月次使用量を更新・保存
    cost = calc_cost(input_tokens, output_tokens)
    usage["input_tokens"]  += input_tokens
    usage["output_tokens"] += output_tokens
    usage["cost_usd"]      += cost
    save_monthly_usage(usage)

    # 5. 予算警告をレビューに付加（有料転換時の安全網）
    _, warn_msg = check_budget(usage)
    full_review = review + warn_msg

    # 6. GitHub PR にコメント投稿
    post_github_comment(repo, pr_number, full_review)


if __name__ == "__main__":
    main()
