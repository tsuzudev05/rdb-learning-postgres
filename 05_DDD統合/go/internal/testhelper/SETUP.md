# testhelper セットアップ手順

DevContainer 内で以下を実行して testcontainers-go を追加する。

```bash
cd 05_DDD統合/go
go get github.com/testcontainers/testcontainers-go@latest
go get github.com/testcontainers/testcontainers-go/modules/postgres@latest
go mod tidy
```

その後、テストを実行：

```bash
go test ./internal/testhelper/...
go test ./infrastructure/repository/...
```

## 注意

- Docker デーモンが起動している必要がある（DevContainer 内では Docker-in-Docker が利用可能）
- テスト起動時に postgres:16-alpine イメージが自動的に pull される（初回は時間がかかる）
- `TruncateAll` は各テストの `t.Cleanup` 内で呼び出すこと
