# AI コードレビュー セットアップ手順

GitHub Actions + Groq（Llama 3.3 70B）で PR を自動レビューする仕組みのセットアップ手順。  
**クレジットカード不要・完全無料**で利用できる。

参考実装: [rdb-learning-postgres](https://github.com/tsuzudev05/rdb-learning-postgres)

---

## 概要

| 項目 | 内容 |
|---|---|
| トリガー | PR の open / synchronize / reopened |
| モデル | Llama 3.3 70B (`llama-3.3-70b-versatile`) |
| 実行基盤 | Groq API |
| コスト | 無料枠 14,400 リクエスト/日・30 RPM（クレカ不要） |
| レビュー言語 | 日本語 |

---

## 必要なファイル

```
<your-repo>/
├── .github/
│   └── workflows/
│       └── ai-review.yml       ← GitHub Actions ワークフロー
└── scripts/
    └── ai_review.py            ← Groq API 呼び出し・PR コメント投稿スクリプト
```

---

## STEP 1: Groq API キーを取得する

1. [console.groq.com](https://console.groq.com) にアクセスしてアカウント登録（Google アカウント可）
2. 左メニュー「API Keys」→「Create API Key」
3. 名前を入力してキーをコピー
4. **このキーは再表示できないため、すぐにメモしておく**

> クレジットカードの登録は不要。

---

## STEP 2: GitHub Secret に登録する

1. GitHub リポジトリページ → **Settings** → **Secrets and variables** → **Actions**
2. 「New repository secret」をクリック
3. 以下を入力して「Add secret」

   | Name | Value |
   |---|---|
   | `GROQ_API_KEY` | STEP 1 で取得した API キー |

> `GITHUB_TOKEN` は GitHub Actions が自動で提供するため、手動登録不要。

---

## STEP 3: ワークフローファイルを push する

```bash
git add .github/workflows/ai-review.yml scripts/ai_review.py
git commit -m "ci: GitHub Actions + Groq（Llama 3.3 70B）による自動 PR レビューを追加"
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

3. **Actions タブ**で「AI Code Review (Groq)」が起動することを確認

4. Actions が成功したら PR の**コメント欄**にレビューが投稿される

   ```
   ## 🤖 AI コードレビュー（Llama 3.3 70B / Groq）
   ### 概要  ...
   ### ✅ 良い点  ...
   ### 🔴 要修正  ...
   ### 🟡 改善提案  ...
   ### 📊 総評  LGTM / 要修正 / 要確認
   ```

5. 確認後、テスト用ブランチと PR はクローズ・削除してよい

---

## トラブルシューティング

### `GROQ_API_KEY` エラー

```
[error] 環境変数が不足しています: ['GROQ_API_KEY']
```

- Secret 名が `GROQ_API_KEY` と完全一致しているか確認
- 登録先が「Repository secrets」か確認

### 429 レート制限

```
groq.RateLimitError: 429
```

- 無料枠の上限（30 RPM / 14,400 req/日）に達した場合に発生
- 個人の PR レビューでは通常発生しない

### diff が空でスキップされる

- PR に実際のコード変更が含まれているか確認
- `*.sum` / `*.lock` / `node_modules/**` は除外対象（正常動作）

---

## コスト見積もり

Groq 無料枠内なら **$0**。

有料プランに移行した場合の参考単価（2025年時点）:

| モデル | 入力 | 出力 |
|---|---|---|
| llama-3.3-70b-versatile | $0.59 / 1M tokens | $0.79 / 1M tokens |

---

## 関連ファイル

- ワークフロー定義: `.github/workflows/ai-review.yml`
- レビュースクリプト: `scripts/ai_review.py`
- GitHub リポジトリ: https://github.com/tsuzudev05/rdb-learning-postgres
