#!/usr/bin/env python3
"""
ai_review.py — PR diff を Gemini Flash に渡してコードレビューを生成し、
               GitHub PR にコメントとして投稿するスクリプト。

コスト制御:
  - 月次トークン累計を .monthly-usage.json に記録（GitHub Actions cache で永続化）
  - 月の累計コストが MONTHLY_BUDGET_USD を超えた場合はスキップし PR にその旨コメント
  - Gemini 2.0 Flash は無料枠（1,500 リクエスト/日）あり。超過時は有料課金。

Usage (GitHub Actions から呼ばれる):
    python scripts/ai_review.py

必要な環境変数:
    GEMINI_API_KEY  Google AI Studio の API キー（GitHub Secret に登録）
    GITHUB_TOKEN    GitHub Actions が自動で提供
    PR_NUMBER       PR 番号
    REPO            "owner/repo" 形式
    PR_TITLE        PR タイトル（プロンプトのコンテキスト用）
    BASE_BRANCH     マージ先ブランチ名
    HEAD_BRANCH     PR のブランチ名
"""

import os
import re
import sys
import json
import urllib.request
import urllib.error
from datetime import datetime, timezone

from google import genai
from google.genai import types

# ─── 設定 ────────────────────────────────────────────────────────────────────

DIFF_FILE            = "diff.txt"
USAGE_FILE           = ".monthly-usage.json"
MAX_DIFF_CHARS       = 400_000     # Gemini 2.0 Flash のコンテキスト上限（1M tokens）を考慮
MODEL                = "gemini-1.5-flash"

# Gemini 1.5 Flash 料金（2025年時点、無料枠超過後）
# 無料枠: 15 RPM・1,500 リクエスト/日（gemini-2.0-flash より広く利用可能）
PRICE_INPUT_PER_MTOK  = 0.075      # USD / million tokens（128K tokens 以下）
PRICE_OUTPUT_PER_MTOK = 0.30       # USD / million tokens（128K tokens 以下）

MONTHLY_BUDGET_USD    = 1.00       # 月の上限（ドル）
BUDGET_WARN_RATIO     = 0.80       # この割合を超えたら警告コメントを付ける

# ─── コスト計算 ──────────────────────────────────────────────────────────────

def calc_cost(input_tokens: int, output_tokens: int) -> float:
    """トークン数から USD コストを計算する（Gemini 1.5 Flash 料金、無料枠超過後）。"""
    return (
        input_tokens  / 1_000_000 * PRICE_INPUT_PER_MTOK
        + output_tokens / 1_000_000 * PRICE_OUTPUT_PER_MTOK
    )


# ─── 月次使用量の永続化 ──────────────────────────────────────────────────────

def current_month() -> str:
    """現在の年月を "YYYY-MM" 形式で返す。"""
    return datetime.now(timezone.utc).strftime("%Y-%m")


def load_monthly_usage() -> dict:
    """月次使用量ファイルを読み込む。月が変わっていたらリセット。"""
    default = {"month": current_month(), "input_tokens": 0, "output_tokens": 0, "cost_usd": 0.0}
    if not os.path.exists(USAGE_FILE):
        return default
    try:
        with open(USAGE_FILE, encoding="utf-8") as f:
            data = json.load(f)
        # 月が変わっていたらリセット
        if data.get("month") != current_month():
            print(f"[info] 月が変わりました（{data.get('month')} → {current_month()}）。使用量をリセットします。")
            return default
        return data
    except (json.JSONDecodeError, KeyError):
        return default


def save_monthly_usage(usage: dict) -> None:
    """月次使用量をファイルに保存する。"""
    with open(USAGE_FILE, "w", encoding="utf-8") as f:
        json.dump(usage, f, ensure_ascii=False, indent=2)
    print(f"[info] 月次使用量を保存しました: ${usage['cost_usd']:.4f} / ${MONTHLY_BUDGET_USD:.2f}")


def check_budget(usage: dict) -> tuple[bool, str]:
    """
    予算チェック。
    戻り値: (続行可能か, 警告メッセージ)
    """
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
## 🤖 AI コードレビュー（Gemini 1.5 Flash）

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
*このレビューは Gemini 1.5 Flash によって自動生成されました。*
"""


# ─── diff 読み込み ────────────────────────────────────────────────────────────

def load_diff() -> str:
    """diff.txt を読み込み、上限文字数で切り詰めて返す。"""
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


# ─── Gemini API 呼び出し ─────────────────────────────────────────────────────

def generate_review(diff: str, pr_title: str, base: str, head: str) -> tuple[str, int, int]:
    """
    Gemini 2.0 Flash を呼び出してレビューを生成する。
    戻り値: (レビュー本文, input_tokens, output_tokens)
    """
    client = genai.Client(api_key=sanitize_api_key(os.environ["GEMINI_API_KEY"]))

    user_message = f"""以下の Pull Request をレビューしてください。

**PR タイトル**: {pr_title}
**ブランチ**: `{head}` → `{base}`

```diff
{diff}
```
"""

    print(f"[info] {MODEL} にリクエスト中...")
    response = client.models.generate_content(
        model=MODEL,
        contents=user_message,
        config=types.GenerateContentConfig(
            system_instruction=SYSTEM_PROMPT,
            max_output_tokens=2048,
        ),
    )

    review        = response.text
    input_tokens  = response.usage_metadata.prompt_token_count or 0
    output_tokens = response.usage_metadata.candidates_token_count or 0
    cost          = calc_cost(input_tokens, output_tokens)

    print(f"[info] 完了 — input: {input_tokens} tok / output: {output_tokens} tok / cost: ${cost:.4f}")
    return review, input_tokens, output_tokens


# ─── GitHub コメント投稿 ──────────────────────────────────────────────────────

def post_github_comment(repo: str, pr_number: str, body: str) -> None:
    """GitHub API を使って PR にコメントを投稿する。"""
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


# ─── メイン ──────────────────────────────────────────────────────────────────

def sanitize_api_key(raw: str) -> str:
    """
    API キーから印字不可能な文字（改行・BOM・制御文字など）を除去する。
    strip() だけでは端以外に混入した制御文字を取り除けないため正規表現で処理する。
    """
    cleaned = re.sub(r"[^\x21-\x7E]", "", raw)  # 印字可能 ASCII (0x21–0x7E) のみ残す
    if not cleaned:
        print("[error] GEMINI_API_KEY が空か、印字可能文字を含みません", file=sys.stderr)
        sys.exit(1)
    if len(raw) != len(cleaned):
        print(f"[info] API キーを sanitize しました（元: {len(raw)} 文字 → 後: {len(cleaned)} 文字）")
    return cleaned


def main() -> None:
    # 環境変数の確認
    required_env = ["GEMINI_API_KEY", "GITHUB_TOKEN", "PR_NUMBER", "REPO"]
    missing = [k for k in required_env if not os.environ.get(k)]
    if missing:
        print(f"[error] 環境変数が不足しています: {missing}", file=sys.stderr)
        sys.exit(1)

    pr_title  = os.environ.get("PR_TITLE", "（タイトルなし）")
    base      = os.environ.get("BASE_BRANCH", "main")
    head      = os.environ.get("HEAD_BRANCH", "feature")
    pr_number = os.environ["PR_NUMBER"]
    repo      = os.environ["REPO"]

    # 1. 月次コスト確認
    usage       = load_monthly_usage()
    can_run, budget_msg = check_budget(usage)

    if not can_run:
        print(f"[warn] 予算超過のためスキップします: {budget_msg}")
        post_github_comment(repo, pr_number, budget_msg)
        sys.exit(0)

    # 2. diff を読み込む
    diff = load_diff()
    print(f"[info] diff: {len(diff)} 文字")

    # 3. Gemini でレビュー生成
    review, input_tokens, output_tokens = generate_review(diff, pr_title, base, head)

    # 4. 月次使用量を更新・保存
    cost             = calc_cost(input_tokens, output_tokens)
    usage["input_tokens"]  += input_tokens
    usage["output_tokens"] += output_tokens
    usage["cost_usd"]      += cost
    save_monthly_usage(usage)

    # 5. 予算警告をレビューに付加（80% 超えた場合）
    _, warn_msg = check_budget(usage)
    full_review = review + warn_msg

    # 6. GitHub PR にコメント投稿
    post_github_comment(repo, pr_number, full_review)


if __name__ == "__main__":
    main()
