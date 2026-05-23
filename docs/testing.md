# テスト手順書

## 概要

DevContainer 内で C++（libpqxx）と Go（pgx）のスモークテストを実行する。  
テスト対象は `05_DDD統合/src/infrastructure/repository/` および `05_DDD統合/go/` の Repository 実装。

---

## 前提条件

- Docker Desktop が起動していること
- VSCode の Dev Containers 拡張機能がインストールされていること

---

## 初回セットアップ

### DevContainer を起動する

```
コマンドパレット（Ctrl+Shift+P）
→「Dev Containers: Rebuild and Reopen in Container」
```

> 初回はイメージのビルドに数分かかります。

`postCreateCommand` により以下が自動実行される：

- Git 設定（`GIT_USER_NAME` / `GIT_USER_EMAIL`）
- PostgreSQL 接続待機
- DDD スキーマ投入（`05_DDD統合/schema.sql`）
- ビルドディレクトリ準備・スクリプトへの実行権限付与

手動で再実行する場合：

```bash
bash /workspace/.devcontainer/setup.sh
```

---

## テストの実行

### 推奨：C++ + Go まとめて実行

```bash
bash /workspace/scripts/test-all.sh
```

### C++ スモークテストのみ

```bash
bash /workspace/scripts/test.sh
```

### Go ビルド・テストのみ

```bash
bash /workspace/scripts/go-test.sh            # ビルド + go test
bash /workspace/scripts/go-test.sh smoke      # + スモークテスト（DB 接続必要）
```

---

## テスト内容

### C++ スモークテスト（`05_DDD統合/src/main.cpp`）

| # | テスト名 | 内容 |
|---|---|---|
| 1 | DB 接続確認 | `current_database()` を取得して接続を確認 |
| 2 | save（新規） | User を INSERT して保存 |
| 3 | findById | ID で User を取得・内容を検証 |
| 4 | save（更新） | 同一 ID で upsert し、名前変更を確認 |
| 5 | findByEmail | メールアドレスで User を取得 |
| 6 | findAll | 全 User 件数が 1 件以上あることを確認 |
| 7 | remove | User を削除し、findById が nullopt を返すことを確認 |
| 8 | ObjectiveRepository 生成 | インスタンス生成のみ確認 |
| 9 | KeyResultRepository 生成 | インスタンス生成のみ確認 |

### Go テスト（`05_DDD統合/go/`）

`go test ./...` で全パッケージを確認。スモークテスト（`cmd/smoke`）は DB 接続が必要。

---

## 成功時の出力

```
========================================
 全体テスト (C++ + Go)
========================================
[事前確認] PostgreSQL 接続確認...
✅ DB 接続 OK

--- C++ ---
[1/2] C++ ビルド中...
✅ ビルド成功: /workspace/05_DDD統合/build/okr_smoke_test
[2/2] スモークテスト実行中...
✅ 全テスト PASSED

--- Go ---
[1/2] go build...
✅ ビルド成功
[2/2] go test...
✅ テスト完了

========================================
 全体テスト 完了
========================================
```

---

## トラブルシューティング

### `relation "users" does not exist` / id が integer になっている

スキーマが古い状態（SERIAL PRIMARY KEY）で残っている場合。`setup.sh` が自動検出して修正するが、手動で解決する場合：

```bash
psql -h postgres -U postgres -d learning -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
psql -h postgres -U postgres -d learning -f /workspace/05_DDD統合/schema.sql
```

### `g++: command not found`

コンテナイメージが古い可能性がある。Rebuild するか、今すぐ使う場合：

```bash
apt-get install -y g++
```

### `go build ./...` でエラー（`pattern ./...`）

Go のモジュールルートに移動してから実行する：

```bash
cd /workspace/05_DDD統合/go
go build ./...
```

または `scripts/go-test.sh` を使うと自動で移動する。

### PostgreSQL に接続できない

```bash
pg_isready -h postgres -U postgres -d learning
# VSCode: Ctrl+Shift+P → "Dev Containers: Rebuild and Reopen in Container"
```

---

## 関連ファイル

```
rdb-learning-postgres/
├── .devcontainer/
│   ├── Dockerfile               # build-essential / libpqxx-dev / Go 1.22
│   ├── docker-compose.yml       # app + postgres サービス定義
│   ├── setup.sh                 # スキーマ投入・UUID 型チェック
│   └── init/01_init_schema.sql  # uuid-ossp 拡張のみ
├── 05_DDD統合/
│   ├── schema.sql               # DB スキーマ（正）
│   ├── src/main.cpp             # C++ スモークテスト本体
│   └── go/cmd/smoke/main.go     # Go スモークテスト本体
└── scripts/
    ├── test-all.sh              # C++ + Go 一括テスト
    ├── test.sh                  # C++ テスト
    └── go-test.sh               # Go テスト
```
