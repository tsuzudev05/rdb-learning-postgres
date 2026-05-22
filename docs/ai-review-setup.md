# AI コードレビュー セットアップ手順

GitHub Actions + Gemini Flash で PR を自動レビューする仕組みのセットアップ手順。  
任意のリポジトリに導入できる。参考実装: [rdb-learning-postgres](https://github.com/tsuzudev05/rdb-learning-postgres)

---

## 概要

| 項目 | 内容 |
|---|---|
| トリガー | PR の open / synchronize / reopened |
| モデル | Gemini 2.0 Flash (`gemini-2.0-flash`) |
| コスト | 無料枠 1,500 リクエスト/日（個人利用なら実質無料） |
| コスト制限 | 月 $1.00 上限（無料枠超過後の安全網） |
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
    └── ai_review.py            ← Gemini API 呼び出し・PR コメント投稿スクリプト
```

参考実装からコピーして使用する:

- ワークフロー: [`.github/workflows/ai-review.yml`](https://github.com/tsuzudev05/rdb-learning-postgres/blob/main/.github/workflows/ai-review.yml)
- スクリプト: [`scripts/ai_review.py`](https://github.com/tsuzudev05/rdb-learning-postgres/blob/main/scripts/ai_review.py)

---

## STEP 1: Gemini API キーを取得する

1. [Google AI Studio](https://aistudio.google.com/app/apikey) にアクセス（Google アカウントでログイン）
2. 「Create API key」をクリック
3. プロジェクトを選択（または新規作成）してキーを生成
4. 表示されたキーをコピーしてメモしておく

> 💡 無料枠: 1,500 リクエスト/日・1,000,000 トークン/分。個人の PR レビュー用途なら無料枠内で収まる。

---

## STEP 2: GitHub Secret に登録する

1. GitHub リポジトリページ → **Settings** → **Secrets and variables** → **Actions**
2. 「New repository secret」をクリック
3. 以下を入力して「Add secret」

   | Name | Value |
   |---|---|
   | `GEMINI_API_KEY` | STEP 1 で取得した API キー |

> `GITHUB_TOKEN` は GitHub Actions が自動で提供するため、手動登録不要。

---

## STEP 3: ワークフローファイルを push する

```bash
git add .github/workflows/ai-review.yml scripts/ai_review.py
git commit -m "ci: GitHub Actions + Gemini Flash による自動 PR レビューを追加"
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

3. **Actions タブ**で「AI Code Review (Gemini)」が起動することを確認

4. Actions が成功したら PR の**コメント欄**に以下の形式でレビューが投稿される

   ```
   ## 🤖 AI コードレビュー（Gemini Flash）
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

### `GEMINI_API_KEY` エラー

```
[error] 環境変数が不足しています: ['GEMINI_API_KEY']
```

- STEP 2 の Secret 名が `GEMINI_API_KEY` と完全一致しているか確認
- Secret の登録先が「Repository secrets」（Organization secrets ではない）か確認

### 月次コスト上限超過

```
⚠️ 月次コスト上限に達しました（$1.0000 / $1.00）
```

- 無料枠（1,500 リクエスト/日）を超えて有料課金が発生した場合に起こる
- 翌月になると自動でリセットされる
- 上限を変更したい場合: `scripts/ai_review.py` の `MONTHLY_BUDGET_USD` を編集

### diff が空でスキップされる

- PR に実際のコード変更が含まれているか確認
- `*.sum` / `*.lock` / `node_modules/**` は除外対象（正常動作）

---

## コスト見積もり

Gemini 2.0 Flash の料金（2025年時点）:

| 項目 | 無料枠 | 有料（超過後） |
|---|---|---|
| リクエスト数 | 1,500 回/日 | — |
| 入力トークン | 1,000,000 tokens/分 | $0.075 / 1M tokens |
| 出力トークン | — | $0.30 / 1M tokens |

PR 1件あたりの目安（diff 約 200 行の場合）:
- 入力: 約 3,000 トークン → $0.000225
- 出力: 約 500 トークン → $0.00015
- **合計: 約 $0.0004/PR（有料時）**

個人の PR レビュー用途なら、**無料枠内で実質 $0** で運用できる。

---

## 関連ファイル（参考実装: rdb-learning-postgres）

- ワークフロー定義: `.github/workflows/ai-review.yml`
- レビュースクリプト: `scripts/ai_review.py`
- GitHub リポジトリ: https://github.com/tsuzudev05/rdb-learning-postgres
