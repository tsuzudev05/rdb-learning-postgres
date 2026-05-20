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

## 実行方法（DevContainer 内）

```bash
cd /workspace/05_DDD統合/go

# 依存解決（初回のみ）
go mod tidy

# スモークテスト実行
go run ./cmd/smoke
```

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
