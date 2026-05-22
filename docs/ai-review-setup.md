# AI コードレビュー セットアップ手順

GitHub Actions + Claude Haiku で PR を自動レビューする仕組みのセットアップ手順。  
任意のリポジトリに導入できる。参考実装: [rdb-learning-postgres](https://github.com/tsuzudev05/rdb-learning-postgres)

---

## 概要

| 項目 | 内容 |
|---|---|
| トリガー | PR の open / synchronize / reopened |
| モデル | Claude Haiku (`claude-haiku-4-5-20251001`) |
| コスト制限 | 月 $1.00 上限（80% 超で警告） |
| レビュー言語 | 日本語 |

---

## 必要なファイル

導入先リポジトリに以下の2ファイルを追加する。

```
<your-repo>/
├── .github/
│   └── workflows/
│       └── ai-review.yml       ← GitHub Actions ワークフロー
└── scripts/
    └── ai_review.py            ← Claude API 呼び出し・PR コメント投稿スクリプト
```

参考実装からコピーして使用する:

- ワークフロー: [`.github/workflows/ai-review.yml`](https://github.com/tsuzudev05/rdb-learning-postgres/blob/main/.github/workflows/ai-review.yml)
- スクリプト: [`scripts/ai_review.py`](https://github.com/tsuzudev05/rdb-learning-postgres/blob/main/scripts/ai_review.py)

---

## STEP 1: Anthropic API キーを取得する

1. [Anthropic Console](https://console.anthropic.com/) にログイン
2. 左メニュー「API Keys」→「Create Key」
3. キー名を入力（例: `github-actions-<your-repo-name>`）してコピー
4. **このキーは再表示できないため、すぐにメモしておく**

> ⚠️ コスト管理: スクリプト内で月 $1.00 の上限を設定済み。超過時は自動でスキップされる。

---

## STEP 2: GitHub Secret に登録する

1. GitHub リポジトリページ → **Settings** → **Secrets and variables** → **Actions**
2. 「New repository secret」をクリック
3. 以下を入力して「Add secret」

   | Name | Value |
   |---|---|
   | `ANTHROPIC_API_KEY` | STEP 1 で取得した API キー |

> `GITHUB_TOKEN` は GitHub Actions が自動で提供するため、手動登録不要。

---

## STEP 3: ワークフローファイルを push する

```bash
git add .github/workflows/ai-review.yml scripts/ai_review.py
git commit -m "ci: GitHub Actions + Claude Haiku による自動 PR レビューを追加"
git push origin main
```

---

## STEP 4: 動作確認

1. 任意のフィーチャーブランチを作成して変更を加える

   ```bash
   git checkout -b test/ai-review-check
   echo "# test" >> README.md
   git add README.md
   git commit -m "test: AI レビュー動作確認用"
   git push origin test/ai-review-check
   ```

2. GitHub で `test/ai-review-check` → `main` の PR を作成

3. **Actions タブ**で「AI Code Review (Claude)」が起動することを確認

4. Actions が成功したら PR の**コメント欄**に以下の形式でレビューが投稿される

   ```
   ## 🤖 AI コードレビュー（Claude Haiku）
   ### 概要
   ...
   ### ✅ 良い点
   ...
   ### 🔴 要修正
   ...
   ### 🟡 改善提案
   ...
   ### 📊 総評
   LGTM / 要修正 / 要確認
   ```

5. 確認後、テスト用ブランチと PR はクローズ・削除してよい

---

## トラブルシューティング

### Actions が起動しない

- `.github/workflows/ai-review.yml` が `main` ブランチに push されているか確認
- ブランチ保護ルールで Actions が無効になっていないか確認

### `ANTHROPIC_API_KEY` エラー

```
[error] 環境変数が不足しています: ['ANTHROPIC_API_KEY']
```

- STEP 2 の Secret 名が `ANTHROPIC_API_KEY` と完全一致しているか確認
- Secret の登録先が「Repository secrets」（Organization secrets ではない）か確認

### 月次コスト上限超過

```
⚠️ 月次コスト上限に達しました（$1.0000 / $1.00）
```

- 翌月になると自動でリセットされる
- 上限を変更したい場合: `scripts/ai_review.py` の `MONTHLY_BUDGET_USD` を編集

### diff が空でスキップされる

- PR に実際のコード変更が含まれているか確認
- `*.sum` / `*.lock` / `node_modules/**` は除外対象（正常動作）

---

## コスト見積もり

Claude Haiku の料金（2025年時点）:

| 項目 | 単価 |
|---|---|
| 入力トークン | $0.80 / 1M tokens |
| 出力トークン | $4.00 / 1M tokens |

PR 1件あたりの目安（diff 約 200 行の場合）:
- 入力: 約 3,000 トークン → 約 $0.0024
- 出力: 約 500 トークン → 約 $0.002
- **合計: 約 $0.004/PR**

月 $1.00 の上限内で **約 250 PR** のレビューが可能。

---

## 関連ファイル（参考実装: rdb-learning-postgres）

- ワークフロー定義: `.github/workflows/ai-review.yml`
- レビュースクリプト: `scripts/ai_review.py`
- GitHub リポジトリ: https://github.com/tsuzudev05/rdb-learning-postgres
