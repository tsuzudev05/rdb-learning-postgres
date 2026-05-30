# OKR Repository — Go (pgx) 実装

C++ 実装（libpqxx）と同じドメインモデル・Repository パターンを Go + pgx v5 で実装したサンプル。

## ディレクトリ構成

```
go/
├── go.mod
├── cmd/
│   └── smoke/
│       └── main.go                  # スモークテスト
├── domain/
│   ├── model/
│   │   └── user/
│   │       ├── user_id.go           # 値オブジェクト: UserId
│   │       ├── email.go             # 値オブジェクト: Email
│   │       └── user.go              # エンティティ: User（集約ルート）
│   └── repository/
│       └── user_repository.go       # インターフェース: UserRepository
└── infrastructure/
    └── repository/
        └── pg_user_repository.go    # pgx 実装: PgUserRepository
```

## 実行方法

### DevContainer（VS Code）

`docker-compose.yml` の `postgres` サービスが自動起動するため、DB の手動セットアップは不要。

```bash
cd /workspace/05_DDD統合/go

make run-api    # API サーバー起動（http://localhost:8080）
make run-smoke  # スモークテスト実行
```

### Claude Code 環境（コンテナ再起動後）

この環境では PostgreSQL サーバーがコンテナ再起動で停止する。**起動のたびに `make init` が必要。**

```bash
cd /workspace/05_DDD統合/go

make init       # PostgreSQL 起動 + DB・スキーマ初期化（冪等）
make run-api    # API サーバー起動（http://localhost:8080）
```

`make init` が行うこと:
- `pg_ctlcluster 16 main start` でローカル PostgreSQL を起動
- `/etc/hosts` に `127.0.0.1 postgres` を追加（未登録の場合のみ）
- `learning` データベースを作成（未作成の場合のみ）
- `schema.sql` を適用（未適用の場合のみ）

## C++ 実装との対応

| C++                         | Go                             |
|-----------------------------|--------------------------------|
| `Result<T>`                 | `(T, error)` の多値返却        |
| `std::optional<T>`          | `*T`（nilポインタ）            |
| `pqxx::connection`          | `pgxpool.Pool`                 |
| `tx.exec_params(...)`       | `pool.Exec(ctx, q, args...)`   |
| `ON CONFLICT DO UPDATE`     | 同じ（SQL は共通）             |

## 設計上のポイント

- ドメイン層（`domain/`）は `pgx` に一切依存しない
- インターフェース（`UserRepository`）はドメイン層に定義し、実装はインフラ層に置く
- `scanUser()` で DB 行 → エンティティの再構築を集約する
- `pgxpool.Pool` を使ってコネクションプールを共有する
