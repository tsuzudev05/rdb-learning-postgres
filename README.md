# rdb-learning-postgres

PostgreSQL を用いた RDB 設計・SQL・パフォーマンスチューニングの学習リポジトリ。  
C++ / Rust / Go × DDD のポートフォリオ開発に向けた、データベース基礎から応用までを体系的にまとめる。

---

## 🎯 学習目的

- C++ × TypeScript ポートフォリオ（OKR 管理ツール）への PostgreSQL 統合
- Go × DDD ポートフォリオ（物流管理アプリ）への PostgreSQL 統合

---

## 🗂️ [WIP] ディレクトリ構成

```
rdb-learning-postgres/
├── .devcontainer/
│   ├── .env.example             # 環境変数サンプル
│   ├── devcontainer-lock.json   # DevContainer ロックファイル
│   ├── devcontainer.json        # VSCode DevContainer 設定
│   ├── docker-compose.yml       # appコンテナ + PostgreSQL 16
│   ├── Dockerfile               # Go / Rust 入り開発環境
│   ├── setup.sh                 # 初回セットアップスクリプト
│   └── init/
│       └── 01_init_schema.sql   # コンテナ初回起動時に自動実行されるスキーマ
├── 02_パフォーマンス/
│   ├── benchmark_index.sh       # インデックス有無でのベンチマークスクリプト
│   └── ベンチマーク結果.md       # インデックス有無での実行時間比較
├── 03_トランザクション/
│   ├── benchmark_deadlock.sh    # デッドロック再現・回避策のベンチマークスクリプト
│   ├── benchmark_isolation.sh   # 分離レベル挙動確認スクリプト
│   ├── benchmark_transaction.sh # BEGIN/ROLLBACK/COMMIT 動作確認スクリプト
│   ├── ACID特性.md              # ACID の解説と実際の動作確認
│   ├── デッドロック動作確認結果.md  # デッドロックの再現と回避策の動作確認結果
│   └── 分離レベル動作確認結果.md   # READ COMMITTED / REPEATABLE READ の挙動確認結果
├── 04_DDD統合/                  # [WIP] DDD × PostgreSQL の実装例
└── 05_実践課題/                 # [WIP] OKR管理・物流管理スキーマ設計
```

---

## 📅 [WIP] 学習ロードマップ

| フェーズ         | 期間     | 内容                                |
| ---------------- | -------- | ----------------------------------- |
| テーブル設計     | 1週目    | 正規化・ER図・スキーマ設計          |
| パフォーマンス   | 2〜3週目 | インデックス・EXPLAIN・クエリ最適化 |
| トランザクション | 3〜4週目 | ACID・分離レベル・デッドロック      |
| DDD統合          | 4〜6週目 | Repository パターン・C++/Go 実装    |
| 実践課題         | 7週目〜  | OKR管理・物流管理スキーマ設計       |

---

## 🛠️ 環境構築（DevContainer）

VSCode の DevContainer を使うことで、PostgreSQL・Go・Rust が揃った開発環境を1コマンドで起動できる。

### 前提

- Docker 26 以上
- VSCode + [Dev Containers 拡張機能](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)

### 起動手順

```bash
# 1. VSCode でリポジトリを開き DevContainer を起動
# コマンドパレット（Cmd+Shift+P）
# →「Dev Containers: Reopen in Container」を選択

# 2. コンテナ起動後、PostgreSQL への接続確認
docker exec -it pg-dev psql -U postgres -d learning
```

### 含まれる環境

| ツール     | バージョン | 用途                         |
| ---------- | ---------- | ---------------------------- |
| PostgreSQL | 16         | RDB 学習・ポートフォリオ統合 |
| Go         | 1.22       | ZOZO向けポートフォリオ開発   |
| Rust       | 最新安定版 | Rust CLI ツール開発          |
| psql       | 16         | PostgreSQL クライアント      |

---

## 📝 学習メモの書き方ルール

各ディレクトリの `.md` ファイルには以下の形式で記録する。

```markdown
## やったこと
## 詰まったこと・解決方法
## 次回やること
```

---

## 🔗 関連リポジトリ（作成予定）

- `okr-manager-cpp` — C++ × TypeScript × PostgreSQL の OKR 管理ツール
- `logistics-ddd-go` — Go × DDD × PostgreSQL の物流管理アプリ
- `rust-cli-tools` — Rust 製 CLI ツール（SQLite 連携）

---

## 📚 参考資料

| 種別 | タイトル                                                       |
| ---- | -------------------------------------------------------------- |
| 動画 | SQL超入門コース 合併版（YouTube）                              |
| 演習 | SQLZOO（https://sqlzoo.net）                                   |
| 公式 | PostgreSQL 16 ドキュメント（https://www.postgresql.org/docs/） |

---
