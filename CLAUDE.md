# CLAUDE.md

Claude Code がこのリポジトリで作業する際に参照するコンテキストファイル。

---

## プロジェクト概要

OKR管理ツール（C++ × TypeScript × PostgreSQL）  
チーム単位でOKRを管理するWebアプリケーション。DDD / クリーンアーキテクチャで実装する。

---

## 開発環境

- OS: Windows（Git Bash）+ VSCode DevContainer
- DevContainer: PostgreSQL 16 / Go 1.22 / Rust / Claude Code
- 改行コード: LF 統一（.gitattributes で設定済み）
- C++ 標準: C++17

---

## アーキテクチャ方針

### DDD / クリーンアーキテクチャ

```
src/
├── common/                         # 共通型（Result<T> など）
├── domain/                         # ドメイン層（外部依存ゼロ）
│   ├── model/                      # 値オブジェクト・エンティティ
│   │   ├── user/
│   │   ├── team/
│   │   ├── period/
│   │   ├── objective/
│   │   └── keyresult/
│   ├── service/                    # ドメインサービス
│   └── repository/                 # Repositoryインターフェース（純粋仮想）
├── application/                    # ユースケース層
│   └── usecase/
├── infrastructure/                 # インフラ層（libpqxx 実装）
│   └── repository/
└── presentation/                   # プレゼンテーション層
    └── cli/
```

### レイヤールール

- `domain/` は外部ライブラリに依存しない（STLのみ）
- `infrastructure/` のみ libpqxx に依存する
- 上位レイヤーから下位レイヤーへの依存のみ許可

---

## 実装方針

### エラーハンドリング

`std::expected`（C++23）の自前実装である `Result<T>` を使用する。例外は使わない。

```cpp
// 成功
Result<UserId>::ok(UserId{value});

// 失敗
Result<UserId>::err("UserId: 無効なUUID形式です: " + value);

// 使用側
auto result = UserId::create("...");
if (!result) { /* result.error() でメッセージ取得 */ }
auto id = result.value();
```

### 値オブジェクト設計方針

- イミュータブル（コンストラクタを private にしてファクトリパターン）
- バリデーションはファクトリ（`create()`）で行い `Result<T>` を返す
- 等価比較演算子（`==` / `!=`）を必ず実装する

---

## 実装状況

### フェーズ1：値オブジェクト ✅

| ファイル                                       | 内容                                  |
| ---------------------------------------------- | ------------------------------------- |
| `common/Result.hpp`                            | `Result<T>` / `Result<void>` 自前実装（in_place構築・void特殊化のok()名前衝突を修正済み） |
| `common/UuidId.hpp`                            | 型安全UUID IDテンプレート `UuidId<Tag>`（フェーズ2で追加） |
| `domain/model/user/UserId.hpp`                 | UUID v4 バリデーション                |
| `domain/model/user/Email.hpp`                  | RFC5322 簡易検証・小文字正規化        |
| `domain/model/team/Role.hpp`                   | `admin` / `member` 列挙型             |
| `domain/model/period/Half.hpp`                 | `H1` / `H2` 列挙型                    |
| `domain/model/period/DateRange.hpp`            | `start < end` 保証                    |
| `domain/model/keyresult/KeyResultProgress.hpp` | `std::variant` による Union型         |

### フェーズ2：エンティティ ✅

- `User` / `Team` / `TeamMember`
- `Period` / `Objective` / `KeyResult` / `KrProgressLog`

### フェーズ3：Repositoryインターフェース ✅

| ファイル                                                  | 内容                                      |
| --------------------------------------------------------- | ----------------------------------------- |
| `domain/repository/IUserRepository.hpp`                   | findById / findByEmail / findAll / save / remove |
| `domain/repository/ITeamRepository.hpp`                   | findById / findByUserId / findAll / save / remove（TeamMember一括管理） |
| `domain/repository/IPeriodRepository.hpp`                 | findById / findByTeamId / findAll / save / remove |
| `domain/repository/IObjectiveRepository.hpp`              | findById / findByPeriodId / findByOwnerId / save / remove |
| `domain/repository/IKeyResultRepository.hpp`              | findById / findByObjectiveId / findByOwnerId / save / remove（KrProgressLog一括管理） |

### フェーズ4：libpqxx 実装 ⬜

- `PgUserRepository` / `PgObjectiveRepository` / `PgKeyResultRepository`

---

## DBスキーマ

- ファイル: `05_DDD統合/schema.sql`
- 設計書: `05_DDD統合/schema.md`
- ドメインモデル: `05_DDD統合/docs/domain-model.md`
- PostgreSQL 16 / uuid-ossp 拡張使用
- `updated_at` はトリガーで自動更新

### 主要テーブル

| テーブル                           | 集約                 |
| ---------------------------------- | -------------------- |
| `users`                            | User 集約ルート      |
| `teams` + `team_members`           | Team 集約            |
| `periods`                          | Period 集約ルート    |
| `objectives`                       | Objective 集約ルート |
| `key_results` + `kr_progress_logs` | KeyResult 集約       |

---

## ユビキタス言語

| 日本語         | 英語        | 説明                      |
| -------------- | ----------- | ------------------------- |
| ユーザー       | User        | システム利用者            |
| チーム         | Team        | OKRを共有するグループ     |
| チームメンバー | TeamMember  | チームに所属するUser      |
| ロール         | Role        | admin / member            |
| 期間           | Period      | 半期単位のOKRサイクル     |
| 目標           | Objective   | 定性的な目標（OのO）      |
| 主要な結果     | KeyResult   | 達成度を測る指標（KR）    |
| 進捗ログ       | ProgressLog | KRの時系列履歴            |
| オーナー       | Owner       | ObjectiveまたはKRの責任者 |

---

## コミット規則

```
feat:  機能追加
fix:   バグ修正
chore: 構成・設定変更（コードなし）
docs:  ドキュメントのみ
test:  テストのみ
```

コミットメッセージは `git commit`（`-m` なし）でVSCodeエディタを開いて記述する。

---

## 関連リンク

- GitHub: https://github.com/tsuzudev05/rdb-learning-postgres
- Zenn: https://zenn.dev/tsuzudev05