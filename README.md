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
├── 01_基礎/
│   ├── 環境構築.md          # Docker による PostgreSQL 起動手順
│   ├── DDL_DML基礎.sql      # CREATE / INSERT / SELECT / UPDATE / DELETE
│   └── JOIN_集計.sql        # INNER JOIN / LEFT JOIN / GROUP BY / サブクエリ
├── 02_テーブル設計/
│   ├── 正規化.md            # 第1〜第3正規形の解説と実例
│   ├── ER図/                # draw.io または Mermaid で作成したER図
│   └── サンプルスキーマ.sql  # 学習用テーブル定義
├── 03_パフォーマンス/
│   ├── インデックス設計.md   # インデックスの種類・貼る基準・EXPLAIN 読み方
│   ├── クエリ最適化.sql     # EXPLAIN ANALYZE を用いたチューニング実例
│   └── ベンチマーク結果.md  # インデックス有無での実行時間比較
├── 04_トランザクション/
│   ├── ACID特性.md          # ACID の解説と実際の動作確認
│   ├── 分離レベル.sql       # READ COMMITTED / REPEATABLE READ の挙動確認
│   └── デッドロック.md      # デッドロックの再現と回避策
├── 05_DDD統合/
│   ├── Repositoryパターン.md # DDD × PostgreSQL の設計方針
│   ├── cpp_example/         # C++ (libpqxx) による Repository 実装例
│   └── go_example/          # Go (pgx) による Repository 実装例
└── 06_実践課題/
    ├── OKR管理スキーマ設計.md # C++ ポートフォリオ向け DB 設計
    └── 物流管理スキーマ設計.md # Go ポートフォリオ向け DB 設計
```

---

## 📅 [WIP] 学習ロードマップ

| フェーズ | 期間 | 内容 |
|---|---|---|
| 基礎 | 1週目 | 環境構築・SELECT/JOIN・正規化 |
| 実践 | 2〜3週目 | インデックス・EXPLAIN・トランザクション・ACID |
| 応用 | 4〜6週目 | DDD × Repository パターン・マイグレーション管理 |
| 統合 | 7週目〜 | C++ / Go ポートフォリオへの PostgreSQL 組み込み |

---

## 🛠️ [WIP] 環境構築（Docker）

```bash
# PostgreSQL コンテナを起動
docker run --name pg-dev \
  -e POSTGRES_PASSWORD=pass \
  -e POSTGRES_DB=learning \
  -p 5432:5432 \
  -d postgres:16

# psql で接続確認
docker exec -it pg-dev psql -U postgres -d learning
```

環境：
- PostgreSQL 16
- Docker 26 以上

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

| 種別 | タイトル |
|---|---|
| 動画 | SQL超入門コース 合併版（YouTube） |
| 演習 | SQLZOO（https://sqlzoo.net） |
| 公式 | PostgreSQL 16 ドキュメント（https://www.postgresql.org/docs/） |
<!-- | 書籍 | 標準SQL＋データベース入門（技術評論社） |
| 書籍 | 達人に学ぶSQL徹底指南書（翔泳社） |
| 書籍 | 内部構造から学ぶPostgreSQL 設計・運用計画の鉄則（技術評論社） | -->

---
