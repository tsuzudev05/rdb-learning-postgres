#!/usr/bin/env python3
"""
auto_pr.py — ブランチの diff を Groq に渡して PR タイトル・本文を生成し、
             GitHub CLI（gh）で PR を自動作成するスクリプト。

Usage (GitHub Actions から呼ばれる):
    python scripts/auto_pr.py

必要な環境変数:
    GROQ_API_KEY   Groq API キー（GitHub Secret に登録）
    GITHUB_TOKEN   GitHub Actions が自動で提供（gh コマンドが使用）
    HEAD_BRANCH    PR のブランチ名（github.ref_name）
    BASE_BRANCH    マージ先ブランチ名（デフォルト: main）
    REPO           "owner/repo" 形式
    COMMIT_LOG     git log --oneline の出力
"""

import os
import sys
import json
import subprocess

from groq import Groq

# ─── 設定 ────────────────────────────────────────────────────────────────────

DIFF_FILE      = "diff.txt"
MAX_DIFF_CHARS = 20_000
MODEL          = "llama-3.3-70b-versatile"

# ─── プロンプト ───────────────────────────────────────────────────────────────

SYSTEM_PROMPT = """\
あなたは Go × C++ × PostgreSQL × DDD（ドメイン駆動設計）プロジェクトの
GitHub PR 説明文ライターです。
渡された git diff とコミット一覧をもとに、
日本語で PR タイトルと本文を生成してください。

出力は必ず以下の JSON 形式のみで返してください（他のテキストは不要）:
{
  "title": "type: 変更の要約（50文字以内）",
  "body": "## 概要\\n...\\n## 変更内容\\n...\\n## テスト方法\\n...\\n## 関連 Issue\\nCloses #XX（不明なら省略）"
}

type の選択基準:
- feat: 新機能・新ファイルの追加
- fix: バグ修正
- docs: ドキュメントのみ
- test: テストのみ
- chore: 設定・ビルド変更
- ci: CI/CD 変更

本文のガイドライン:
- 概要: 何をなぜ変更したか 3〜5 行
- 変更内容: ファイル名を backtick で囲んで箇条書き
- 実装の設計ポイント: DDD / クリーンアーキテクチャ観点で重要な判断があれば記載（なければ省略）
- テスト方法: DevContainer が必要な場合はその旨を明記
- 関連 Issue: コミットメッセージに (#XX) があれば "Closes #XX" と記載
"""

def build_user_prompt(commit_log: str, diff: str) -> str:
    diff_truncated = diff[:MAX_DIFF_CHARS]
    if len(diff) > MAX_DIFF_CHARS:
        diff_truncated += f"\n\n... （差分が長いため {len(diff) - MAX_DIFF_CHARS} 文字を省略）"
    return f"""\
## コミット一覧
{commit_log}

## git diff（main との差分）
```diff
{diff_truncated}
```
"""

# ─── Groq 呼び出し ────────────────────────────────────────────────────────────

def generate_pr_description(commit_log: str, diff: str) -> tuple[str, str]:
    client = Groq(api_key=os.environ["GROQ_API_KEY"])
    response = client.chat.completions.create(
        model=MODEL,
        messages=[
            {"role": "system", "content": SYSTEM_PROMPT},
            {"role": "user",   "content": build_user_prompt(commit_log, diff)},
        ],
        temperature=0.2,
        max_tokens=1500,
    )

    content = response.choices[0].message.content.strip()

    # JSON 部分だけ抽出（```json ... ``` で囲まれている場合も対応）
    if "```" in content:
        import re
        m = re.search(r"```(?:json)?\s*(\{.*?\})\s*```", content, re.DOTALL)
        if m:
            content = m.group(1)

    data = json.loads(content)
    return data["title"], data["body"]

# ─── gh pr create ─────────────────────────────────────────────────────────────

def create_pr(title: str, body: str, head: str, base: str) -> None:
    result = subprocess.run(
        [
            "gh", "pr", "create",
            "--title", title,
            "--body",  body,
            "--head",  head,
            "--base",  base,
        ],
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        # 既に PR が存在する場合はスキップ（エラーではない）
        if "already exists" in result.stderr or "already exists" in result.stdout:
            print(f"ℹ️  PR already exists for branch '{head}' — skipping.")
            return
        print(f"❌ gh pr create failed:\n{result.stderr}", file=sys.stderr)
        sys.exit(1)
    print(f"✅ PR created: {result.stdout.strip()}")

# ─── main ─────────────────────────────────────────────────────────────────────

def main() -> None:
    head_branch = os.environ.get("HEAD_BRANCH", "")
    base_branch = os.environ.get("BASE_BRANCH", "main")
    commit_log  = os.environ.get("COMMIT_LOG", "(コミット情報なし)")

    if not head_branch:
        print("❌ HEAD_BRANCH が設定されていません", file=sys.stderr)
        sys.exit(1)

    # diff を読み込む
    try:
        with open(DIFF_FILE, encoding="utf-8") as f:
            diff = f.read()
    except FileNotFoundError:
        diff = "(差分ファイルが見つかりません)"

    if not diff.strip():
        print("⚠️  diff が空です。PR の作成をスキップします。")
        return

    print(f"🤖 Generating PR description with Groq ({MODEL})...")
    title, body = generate_pr_description(commit_log, diff)
    print(f"\n📝 Title: {title}")

    create_pr(title, body, head_branch, base_branch)


if __name__ == "__main__":
    main()
