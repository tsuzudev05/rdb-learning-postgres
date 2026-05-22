#!/usr/bin/env python3
"""
ai_review.py — PR diff を Groq（Llama 3.3 70B）に渡してコードレビューを生成し、
               GitHub PR にインラインコメント + サマリーとして投稿するスクリプト。

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
MAX_DIFF_CHARS = 25_000   # Groq 無料枠 12,000 TPM 制限に合わせた上限
MODEL          = "llama-3.3-70b-versatile"

PRICE_INPUT_PER_MTOK  = 0.59
PRICE_OUTPUT_PER_MTOK = 0.79

MONTHLY_BUDGET_USD = 1.00
BUDGET_WARN_RATIO  = 0.80


# ─── コスト計算 ──────────────────────────────────────────────────────────────

def calc_cost(input_tokens: int, output_tokens: int) -> float:
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
        return False, (
            f"⚠️ **月次コスト上限に達しました**（${current_cost:.4f} / ${MONTHLY_BUDGET_USD:.2f}）。"
            "AIレビューをスキップしました。来月リセットされます。"
        )

    warn = ""
    if ratio >= BUDGET_WARN_RATIO:
        warn = (
            f"\n\n> ⚠️ 今月の使用コスト: **${current_cost:.4f}** / ${MONTHLY_BUDGET_USD:.2f} "
            f"（残り ${remaining:.4f}）"
        )
    return True, warn


# ─── diff パーサー ────────────────────────────────────────────────────────────

def parse_diff(diff: str) -> dict[str, list[int]]:
    """
    diff から変更されたファイルと追加行の行番号を抽出する。
    戻り値: {ファイルパス: [変更後の行番号, ...]}
    インラインコメントは変更行のみに付けられるため、この情報を AI に渡す。
    """
    changed: dict[str, list[int]] = {}
    current_file: str | None = None
    current_line = 0

    for line in diff.split("\n"):
        if line.startswith("diff --git "):
            # "diff --git a/foo.go b/foo.go" → "foo.go"
            parts = line.split(" b/", 1)
            current_file = parts[1].strip() if len(parts) > 1 else None
            if current_file:
                changed.setdefault(current_file, [])

        elif line.startswith("@@ ") and current_file:
            # "@@ -old_start,old_count +new_start,new_count @@"
            m = re.search(r"\+(\d+)", line)
            current_line = int(m.group(1)) if m else 0

        elif current_file:
            if line.startswith("+++") or line.startswith("---"):
                pass  # ヘッダー行は無視
            elif line.startswith("+"):
                changed[current_file].append(current_line)
                current_line += 1
            elif line.startswith("-"):
                pass  # 削除行は新ファイルの行番号に影響しない
            elif not line.startswith("\\"):
                current_line += 1  # コンテキスト行

    # 変更行が1つもないファイルは除外
    return {f: lines for f, lines in changed.items() if lines}


# ─── プロンプト ───────────────────────────────────────────────────────────────

def build_system_prompt(changed_files: dict[str, list[int]]) -> str:
    """変更ファイルと行番号の情報を含むシステムプロンプトを生成する。"""
    file_info_lines = []
    for path, lines in changed_files.items():
        if len(lines) > 20:
            file_info_lines.append(f"  {path}: 行 {lines[:20]}... (他 {len(lines)-20} 行)")
        else:
            file_info_lines.append(f"  {path}: 行 {lines}")
    file_info = "\n".join(file_info_lines) if file_info_lines else "  （変更ファイルなし）"

    return f"""あなたは DDD・C++17・Go・PostgreSQL に詳しいシニアエンジニアです。
PR の diff をレビューし、以下の JSON スキーマに従って **JSON のみ** を出力してください。
前置き・説明文・コードブロック記号（```）は一切含めないこと。

JSONスキーマ:
- "summary": 文字列（改行は \\n で表現）。レビュー全体をMarkdownで記述。
- "comments": 配列。各要素は {{"path": 文字列, "line": 整数, "body": 文字列}}。

summaryの内容テンプレート（このまま使ってよい）:
## 🤖 AI コードレビュー（Llama 3.3 70B / Groq）

### 概要
（2〜3行で要約）

### ✅ 良い点
- （箇条書き）

### 🔴 要修正
（なければ「なし」）

### 🟡 改善提案
- （任意）

### 📊 総評
LGTM / 要修正 / 要確認

---
*このレビューは Llama 3.3 70B（Groq）によって自動生成されました。*

commentsのルール:
- 具体的なコードの問題を指摘する場合のみ記載（全体的な指摘はsummaryに書く）
- pathとlineは必ず下記「変更ファイル一覧」に含まれる値のみ使用すること
- 指摘がない場合は空配列 [] にすること

変更ファイル一覧:
{file_info}

レビュー観点: コードの正確性・バグ・DDD原則への準拠・C++17/Goイディオム・セキュリティ・パフォーマンス・テスト"""


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

def extract_json_from_response(text: str) -> dict:
    """
    AIレスポンスから JSON を取り出して正規化する。
    - コードブロック（```json ... ```）を除去
    - summary が配列になっていた場合は結合して文字列に修正
    - summary が JSON の複数キーに分散していた場合も復元を試みる
    """
    # コードブロックを除去
    cleaned = re.sub(r"^```(?:json)?\s*", "", text.strip(), flags=re.MULTILINE)
    cleaned = re.sub(r"\s*```$", "", cleaned.strip(), flags=re.MULTILINE)

    data = json.loads(cleaned.strip())

    # summary の正規化
    summary = data.get("summary", "")
    if isinstance(summary, list):
        # ["## header", "\n\n### section", ...] → 結合
        summary = "".join(str(s) for s in summary)
        data["summary"] = summary
    elif not isinstance(summary, str):
        data["summary"] = str(summary)

    # Llama がセクションを別キーに分散させた場合の復元
    # 例: {summary: "## title", "\n\n### 概要\n": "内容", ...}
    if summary and not any(h in summary for h in ["### 概要", "### 良い点", "### ✅"]):
        extra_parts = []
        for key, val in data.items():
            if key in ("summary", "comments"):
                continue
            if isinstance(val, list):
                val = "\n".join(f"- {v}" for v in val if isinstance(v, str))
            if key.strip() or val:
                extra_parts.append(f"{key}{val}")
        if extra_parts:
            data["summary"] = summary + "".join(extra_parts)

    # comments の正規化（配列でない場合は空に）
    if not isinstance(data.get("comments"), list):
        data["comments"] = []

    return data


def generate_review(
    diff: str,
    pr_title: str,
    base: str,
    head: str,
    changed_files: dict[str, list[int]],
) -> tuple[str, list[dict], int, int]:
    """
    Groq（Llama 3.3 70B）を呼び出してレビューを生成する。
    戻り値: (サマリー, インラインコメントリスト, input_tokens, output_tokens)
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
            {"role": "system", "content": build_system_prompt(changed_files)},
            {"role": "user", "content": user_message},
        ],
        model=MODEL,
        max_tokens=2048,
    )

    raw_text     = chat_completion.choices[0].message.content
    input_tokens  = chat_completion.usage.prompt_tokens
    output_tokens = chat_completion.usage.completion_tokens
    cost          = calc_cost(input_tokens, output_tokens)

    print(f"[info] 完了 — input: {input_tokens} tok / output: {output_tokens} tok / 参考コスト: ${cost:.4f}")

    # JSON をパース（失敗時はサマリーのみ返す）
    try:
        result   = extract_json_from_response(raw_text)
        summary  = result.get("summary", raw_text)
        comments = result.get("comments", [])
        print(f"[info] インラインコメント候補: {len(comments)} 件")
    except (json.JSONDecodeError, ValueError) as e:
        print(f"[warn] JSON パース失敗（{e}）。サマリーのみ投稿します。")
        summary  = raw_text
        comments = []

    return summary, comments, input_tokens, output_tokens


# ─── GitHub API 投稿 ──────────────────────────────────────────────────────────

def _github_request(url: str, body: dict, token: str) -> dict:
    data    = json.dumps(body).encode("utf-8")
    headers = {
        "Authorization": f"Bearer {token}",
        "Accept": "application/vnd.github+json",
        "Content-Type": "application/json",
        "X-GitHub-Api-Version": "2022-11-28",
    }
    req = urllib.request.Request(url, data=data, headers=headers, method="POST")
    try:
        with urllib.request.urlopen(req) as resp:
            return json.loads(resp.read())
    except urllib.error.HTTPError as e:
        body_text = e.read().decode()
        print(f"[error] GitHub API エラー: {e.code} {body_text}", file=sys.stderr)
        raise


def post_pr_review(
    repo: str,
    pr_number: str,
    summary: str,
    inline_comments: list[dict],
    changed_files: dict[str, list[int]],
    token: str,
) -> None:
    """
    PR Review API でサマリー + インラインコメントを投稿する。
    存在しないファイル・行番号はスキップして失敗を防ぐ。
    """
    # 変更行セットを作成してバリデーション
    valid_comments = []
    skipped = 0
    for c in inline_comments:
        path = c.get("path", "")
        line = c.get("line")
        body = c.get("body", "")

        if not path or not isinstance(line, int) or not body:
            skipped += 1
            continue

        allowed_lines = changed_files.get(path, [])
        if line not in allowed_lines:
            print(f"[warn] スキップ: {path}:{line} は変更行に含まれていません")
            skipped += 1
            continue

        valid_comments.append({"path": path, "line": line, "side": "RIGHT", "body": body})

    if skipped:
        print(f"[info] {skipped} 件のコメントをスキップしました（無効な path/line）")
    print(f"[info] インラインコメント投稿: {len(valid_comments)} 件")

    url = f"https://api.github.com/repos/{repo}/pulls/{pr_number}/reviews"
    payload = {
        "body": summary,
        "event": "COMMENT",
        "comments": valid_comments,
    }

    try:
        result = _github_request(url, payload, token)
        print(f"[info] レビュー投稿完了: {result.get('html_url', '（URL取得失敗）')}")
    except urllib.error.HTTPError:
        # インラインコメント付きで失敗した場合はサマリーのみ再投稿
        print("[warn] インラインコメント付き投稿が失敗しました。サマリーのみ再投稿します。")
        fallback_url = f"https://api.github.com/repos/{repo}/issues/{pr_number}/comments"
        result = _github_request(fallback_url, {"body": summary}, token)
        print(f"[info] サマリーのみ投稿完了: {result.get('html_url', '')}")


# ─── ユーティリティ ───────────────────────────────────────────────────────────

def sanitize_api_key(raw: str) -> str:
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
    token     = os.environ["GITHUB_TOKEN"]

    # 1. 月次コスト確認
    usage = load_monthly_usage()
    can_run, budget_msg = check_budget(usage)
    if not can_run:
        print(f"[warn] 予算超過のためスキップします")
        fallback_url = f"https://api.github.com/repos/{repo}/issues/{pr_number}/comments"
        _github_request(fallback_url, {"body": budget_msg}, token)
        sys.exit(0)

    # 2. diff を読み込む
    diff = load_diff()
    print(f"[info] diff: {len(diff)} 文字")

    # 3. diff をパースして変更ファイル・行番号を取得
    changed_files = parse_diff(diff)
    print(f"[info] 変更ファイル: {len(changed_files)} 件")

    # 4. Groq でレビュー生成
    summary, inline_comments, input_tokens, output_tokens = generate_review(
        diff, pr_title, base, head, changed_files
    )

    # 5. 月次使用量を更新・保存
    cost = calc_cost(input_tokens, output_tokens)
    usage["input_tokens"]  += input_tokens
    usage["output_tokens"] += output_tokens
    usage["cost_usd"]      += cost
    save_monthly_usage(usage)

    # 6. 予算警告を summary に付加
    _, warn_msg = check_budget(usage)
    summary += warn_msg

    # 7. PR にインラインコメント + サマリーを投稿
    post_pr_review(repo, pr_number, summary, inline_comments, changed_files, token)


if __name__ == "__main__":
    main()
