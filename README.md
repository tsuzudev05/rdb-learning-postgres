# rdb-learning-postgres

C++ × Go × PostgreSQL × DDD のポートフォリオ開発リポジトリ。  
OKR 管理ツールを題材に、クリーンアーキテクチャ・DDD・RDB 設計を実践する。

---

## 🗺️ どこを見ればいい？

| 知りたいこと | 参照先 |
|---|---|
| プロジェクト概要・実装状況 | [`CLAUDE.md`](./CLAUDE.md) |
| 毎日の作業手順・コミット規則 | [`DAILY_WORKFLOW.md`](./DAILY_WORKFLOW.md) |
| テストの実行方法 | [`docs/testing.md`](./docs/testing.md) |
| DB スキーマ設計の解説 | [`05_DDD統合/schema.md`](./05_DDD統合/schema.md) |
| ドメインモデル・集約設計 | [`05_DDD統合/docs/domain-model.md`](./05_DDD統合/docs/domain-model.md) |
| Go 実装の概要 | [`05_DDD統合/go/README.md`](./05_DDD統合/go/README.md) |
| AI レビューの設定方法 | [`docs/ai-review-setup.md`](./docs/ai-review-setup.md) |
| 実装バックログ（Issues） | [GitHub Issues](https://github.com/tsuzudev05/rdb-learning-postgres/issues) |

---

## 📦 技術スタック

| レイヤー | 技術 |
|---|---|
| ドメイン・インフラ（C++） | C++17 / libpqxx / PostgreSQL 16 |
| インフラ（Go） | Go 1.22 / pgx v5 / PostgreSQL 16 |
| DB | PostgreSQL 16 / uuid-ossp |
| 開発環境 | VSCode DevContainer / Docker |
| CI | GitHub Actions（AI レビュー：Groq） |

---

## 🏗️ アーキテクチャ

DDD / クリーンアーキテクチャで実装。

```
src/
├── common/           # 共通型（Result<T> など）
├── domain/           # ドメイン層（外部依存ゼロ）
│   ├── model/        # 値オブジェクト・エンティティ
│   ├── service/      # ドメインサービス
│   └── repository/   # Repository インターフェース（純粋仮想）
├── application/      # ユースケース層
│   └── usecase/
├── infrastructure/   # インフラ層（libpqxx / pgx 実装）
│   └── repository/
└── presentation/     # プレゼンテーション層
    └── cli/
```

---

## 📈 実装フェーズ

| フェーズ | 内容 | 状態 |
|---|---|---|
| 1 | 値オブジェクト（UserId / Email / Role / DateRange など） | ✅ 完了 |
| 2 | エンティティ（User / Team / Period / Objective / KeyResult） | ✅ 完了 |
| 3 | Repository インターフェース（I*Repository） | ✅ 完了 |
| 4 | libpqxx 実装（PgUserRepository など） | ✅ 完了 |
| 5 | Go (pgx) 実装（UserRepository） | ✅ 完了 |
| 6 | ユースケース層（C++）実装 | 🔄 進行中 |

---

## 🚀 環境構築

### 前提

- Docker Desktop（26 以上）
- VSCode + [Dev Containers 拡張](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)

### 起動

```
コマンドパレット（Ctrl+Shift+P）
→「Dev Containers: Rebuild and Reopen in Container」
```

`postCreateCommand` により以下が自動実行される：
- Git 設定
- PostgreSQL 接続待機
- DDD スキーマ投入（`05_DDD統合/schema.sql`）
- ビルドディレクトリ準備

### テスト実行（起動後）

```bash
bash scripts/test-all.sh    # C++ + Go をまとめて確認
bash scripts/test.sh        # C++ スモークテストのみ
bash scripts/go-test.sh     # Go ビルド・テストのみ
```

詳細は [`docs/testing.md`](./docs/testing.md) を参照。

---

## 🗂️ ディレクトリ構成

```
rdb-learning-postgres/
├── .devcontainer/
│   ├── Dockerfile               # Go 1.22 / build-essential / libpqxx-dev
│   ├── docker-compose.yml       # app コンテナ + PostgreSQL 16
│   ├── setup.sh                 # postCreateCommand（スキーマ投入・Git 設定）
│   └── init/
│       └── 01_init_schema.sql   # uuid-ossp 拡張のみ（テーブルは setup.sh で投入）
├── 02_パフォーマンス/            # インデックス・ベンチマーク学習ノート
├── 03_トランザクション/          # ACID・分離レベル・デッドロック学習ノート
├── 05_DDD統合/
│   ├── schema.sql               # DB スキーマ（正）
│   ├── schema.md                # スキーマ設計解説
│   ├── docs/domain-model.md     # ドメインモデル・集約設計
│   ├── src/                     # C++ 実装（domain / infrastructure）
│   ├── go/                      # Go 実装（domain / infrastructure）
│   └── build/                   # C++ ビルド成果物（.gitignore）
├── docs/
│   ├── testing.md               # テスト手順書
│   ├── ai-review-setup.md       # GitHub Actions AI レビュー設定手順
│   └── github-issues-backlog.md # Issues 管理ルール
├── scripts/
│   ├── build.sh                 # C++ ビルドのみ
│   ├── test.sh                  # C++ スモークテスト
│   ├── go-test.sh               # Go ビルド・テスト
│   └── test-all.sh              # C++ + Go 一括テスト
├── CLAUDE.md                    # プロジェクト概要・実装状況（Claude Code 用）
├── DAILY_WORKFLOW.md            # 毎日の作業手順書
└── README.md                    # 本ファイル
```

---

## 🔗 関連リンク

- GitHub: https://github.com/tsuzudev05/rdb-learning-postgres
- Zenn: https://zenn.dev/tsuzudev05
