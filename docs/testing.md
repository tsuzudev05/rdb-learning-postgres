# テスト手順書

## 概要

このプロジェクトでは DevContainer 内で C++ スモークテストを実行する。  
テスト対象は `05_DDD統合/src/infrastructure/repository/` の libpqxx 実装（フェーズ4）。

---

## 前提条件

- Docker Desktop が起動していること
- VSCode の Dev Containers 拡張機能がインストールされていること

---

## 初回セットアップ

### 1. DevContainer を起動する

VSCode でプロジェクトを開き、コマンドパレット（`Ctrl+Shift+P`）から実行：

```
Dev Containers: Rebuild and Reopen in Container
```

> 初回はイメージのビルドに数分かかります。

### 2. セットアップスクリプトの確認

`postCreateCommand` により自動実行されるが、手動で再実行する場合：

```bash
bash /workspace/.devcontainer/setup.sh
```

セットアップが完了すると以下が確認できる：

```
✅ Git設定完了
✅ PostgreSQL に接続できました
✅ DDD スキーマ投入完了
✅ C++ ビルドディレクトリ準備完了

📌 インストール済みツール：
  g++     : g++ (Ubuntu ...) 13.x.x
  make    : GNU Make 4.x
  libpqxx : 7.x.x (または 6.4.5)
  ...
```

---

## テストの実行

### 方法 A — テストスクリプト（推奨）

```bash
bash /workspace/scripts/test.sh
```

### 方法 B — Makefile を直接使う

```bash
cd /workspace/05_DDD統合
make test
```

### 方法 C — ステップごとに実行

```bash
cd /workspace/05_DDD統合

# ビルドのみ
make build

# ビルド済みバイナリを直接実行
DATABASE_URL=postgresql://postgres:pass@postgres:5432/learning \
  ./build/okr_smoke_test
```

---

## テスト内容

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

---

## 成功時の出力

```
========================================
 C++ スモークテスト
========================================
[事前確認] PostgreSQL 接続確認...
✅ DB 接続 OK

[1/2] C++ ビルド中...
✅ ビルド成功: ./build/okr_smoke_test

[2/2] スモークテスト実行中...
========================================
 OKR Repository スモークテスト
========================================
[1] DB 接続確認... ✅  DB=learning
[2] save (新規)... ✅
[3] findById... ✅  name=スモークテスト太郎
[4] save (更新)... ✅  name=スモークテスト更新済み
[5] findByEmail... ✅
[6] findAll... ✅  count=1
[7] remove... ✅
[8] PgObjectiveRepository 生成... ✅
[9] PgKeyResultRepository 生成... ✅
========================================
 ✅ 全テスト PASSED
========================================
```

---

## トラブルシューティング

### PostgreSQL に接続できない

```bash
# コンテナの起動状態を確認
pg_isready -h postgres -U postgres -d learning

# DevContainer を再起動
# VSCode: Ctrl+Shift+P → "Dev Containers: Rebuild and Reopen in Container"
```

### コンパイルエラー

```bash
# libpqxx のバージョン確認
pkg-config --modversion libpqxx

# g++ のバージョン確認
g++ --version
```

### ビルドディレクトリが存在しない

```bash
mkdir -p /workspace/05_DDD統合/build
```

---

## ファイル構成

```
rdb-learning-postgres/
├── .devcontainer/
│   ├── Dockerfile          # build-essential / libpqxx-dev / libpq-dev を含む
│   ├── docker-compose.yml  # app + postgres サービス定義
│   └── setup.sh            # 初回セットアップスクリプト
├── 05_DDD統合/
│   ├── Makefile            # make build / make test
│   ├── schema.sql          # DB スキーマ（初回セットアップ時に投入）
│   └── src/
│       ├── main.cpp        # スモークテスト本体
│       └── infrastructure/
│           └── repository/
│               ├── PgUserRepository.hpp
│               ├── PgObjectiveRepository.hpp
│               └── PgKeyResultRepository.hpp
├── docs/
│   └── testing.md          # 本ファイル
└── scripts/
    └── test.sh             # テスト実行スクリプト
```
