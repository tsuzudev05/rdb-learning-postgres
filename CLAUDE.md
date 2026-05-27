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

### フェーズ4：libpqxx 実装 ✅

| ファイル                                                      | 内容                                      |
| ------------------------------------------------------------- | ----------------------------------------- |
| `infrastructure/repository/PgUserRepository.hpp`              | findById / findByEmail / findAll / save（upsert）/ remove |
| `infrastructure/repository/PgTeamRepository.hpp`              | findById / findByUserId / findAll / save（upsert + TeamMember 一括置換）/ remove |
| `infrastructure/repository/PgPeriodRepository.hpp`            | findById / findByTeamId / findAll / save（upsert）/ remove |
| `infrastructure/repository/PgObjectiveRepository.hpp`         | findById / findByPeriodId / findByOwnerId / save（upsert）/ remove |
| `infrastructure/repository/PgKeyResultRepository.hpp`         | findById / findByObjectiveId / findByOwnerId / save（KeyResult + KrProgressLog 一括）/ remove |

### フェーズ5：Go (pgx) 実装 ✅

配置先: `05_DDD統合/go/`

| ファイル                                                      | 内容                                      |
| ------------------------------------------------------------- | ----------------------------------------- |
| `go.mod`                                                      | Go モジュール（pgx v5 依存）              |
| `domain/model/user/user_id.go`                                | 値オブジェクト UserId（UUID v4バリデーション） |
| `domain/model/user/email.go`                                  | 値オブジェクト Email（正規化・バリデーション） |
| `domain/model/user/user.go`                                   | エンティティ User（集約ルート）           |
| `domain/repository/user_repository.go`                        | インターフェース UserRepository           |
| `infrastructure/repository/pg_user_repository.go`             | FindByID / FindByEmail / FindAll / Save（upsert）/ Remove |
| `cmd/smoke/main.go`                                           | スモークテスト（C++ 版と同等の CRUD 確認）|

**C++ との主な対応:**
- `Result<T>` → `(T, error)` の多値返却
- `std::optional<T>` → `*T`（nilポインタ）
- `pqxx::connection` → `pgxpool.Pool`

### フェーズ5拡張：Go (pgx) Team / Period / Objective / KeyResult 実装 ✅

配置先: `05_DDD統合/go/`

| ファイル                                                        | 内容                                      |
| --------------------------------------------------------------- | ----------------------------------------- |
| `domain/model/team/team_id.go`                                  | TeamId 値オブジェクト                    |
| `domain/model/team/team_member_id.go`                           | TeamMemberId 値オブジェクト              |
| `domain/model/team/role.go`                                     | Role 値オブジェクト（admin / member）     |
| `domain/model/team/team_member.go`                              | TeamMember エンティティ（Team集約内）    |
| `domain/model/team/team.go`                                     | Team 集約ルート（AddMember / RemoveMember / ChangeMemberRole）|
| `domain/model/period/period_id.go`                              | PeriodId 値オブジェクト                  |
| `domain/model/period/half.go`                                   | Half 値オブジェクト（H1 / H2）            |
| `domain/model/period/date_range.go`                             | DateRange 値オブジェクト（start < end 保証）|
| `domain/model/period/period.go`                                 | Period 集約ルート                         |
| `domain/model/objective/objective_id.go`                        | ObjectiveId 値オブジェクト               |
| `domain/model/objective/objective.go`                           | Objective 集約ルート                      |
| `domain/model/keyresult/key_result_id.go`                       | KeyResultId 値オブジェクト               |
| `domain/model/keyresult/kr_progress_log_id.go`                  | KrProgressLogId 値オブジェクト           |
| `domain/model/keyresult/kr_progress_log.go`                     | KrProgressLog エンティティ               |
| `domain/model/keyresult/key_result.go`                          | KeyResult 集約ルート（numeric / checkbox）|
| `domain/repository/team_repository.go`                          | TeamRepository インターフェース           |
| `domain/repository/period_repository.go`                        | PeriodRepository インターフェース         |
| `domain/repository/objective_repository.go`                     | ObjectiveRepository インターフェース      |
| `domain/repository/key_result_repository.go`                    | KeyResultRepository インターフェース      |
| `infrastructure/repository/pg_team_repository.go`               | upsert + team_members 全置換             |
| `infrastructure/repository/pg_period_repository.go`             | FindByID / FindByTeamID / FindAll / Save / Remove |
| `infrastructure/repository/pg_objective_repository.go`          | FindByID / FindByPeriodID / FindByOwnerID / Save / Remove |
| `infrastructure/repository/pg_key_result_repository.go`         | upsert KR + append-only progress logs    |
| `cmd/smoke/main.go`                                             | スモークテスト拡張（テスト [1]-[26]）     |

**要手動確認（DevContainer 内）:**
- `bash /workspace/scripts/go-test.sh smoke` でビルド＋スモークテスト実行

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
### フェーズ6：ユースケース層（C++）✅

配置先: `05_DDD統合/src/application/usecase/`

| ファイル                                       | 内容                                  |
| ---------------------------------------------- | ------------------------------------- |
| `application/usecase/UserUseCase.hpp`          | CreateUser / GetUser / ListUsers / DeleteUser（DI パターン、Result<T> で統一）|
| `application/usecase/TeamUseCase.hpp`          | CreateTeam / GetTeam / ListTeams / ListTeamsByUser / DeleteTeam / AddMember / RemoveMember / ChangeMemberRole |
| `application/usecase/PeriodUseCase.hpp`        | CreatePeriod / GetPeriod / ListPeriods / ListPeriodsByTeam / DeletePeriod |
| `application/usecase/ObjectiveUseCase.hpp`     | CreateObjective / GetObjective / ListObjectivesByPeriod / ListObjectivesByOwner / DeleteObjective |
| `application/usecase/KeyResultUseCase.hpp`     | CreateNumericKeyResult / CreateCheckboxKeyResult / GetKeyResult / ListKeyResultsByObjective / ListKeyResultsByOwner / UpdateNumericProgress / UpdateCheckboxProgress / DeleteKeyResult |

**設計ポイント:**
- 全 Repository を `shared_ptr` で DI → インフラ層への依存ゼロ
- 入力 DTO と出力 DTO を分離（UseCaseOutput::from() ファクトリパターン）
- `CreateUser` にメールアドレス重複チェックを組み込み
- Delete 系はすべて冪等（存在しない場合もエラーにしない）
- `TeamUseCase` の AddMember / RemoveMember / ChangeMemberRole は Read-Modify-Write パターン
- `KeyResultUseCase` の UpdateProgress は Read-Modify-Write パターン（進捗種別チェック付き）
- KR 作成を numeric / checkbox で別メソッドに分離（型安全な DTO 設計）
- `UuidId<Tag>::generate()` を共通テンプレートに追加（全 ID 型で共有）

**要手動確認（DevContainer 内）:**
- `scripts/check-compile.sh` でコンパイル確認
- スモークテストへの UseCase 統合確認

### フェーズ7：CLI プレゼンテーション層（C++）✅

配置先: `05_DDD統合/src/presentation/cli/`

| ファイル                                       | 内容                                  |
| ---------------------------------------------- | ------------------------------------- |
| `presentation/cli/CliApp.hpp`                  | 対話型 CLI（メインループ・コマンドディスパッチ・全 UseCase の DI） |
| `src/main_cli.cpp`                             | CLI エントリーポイント（DB接続 → Repository → UseCase → CliApp）|
| `Makefile`（更新）                              | `cli` / `run-cli` ターゲットを追加 |

**対応コマンド:**
- `user create/list/get/delete`
- `team create/list/get/add-member`
- `period create/list/list-by-team/get/delete`
- `objective create/list/get/delete`
- `help` / `quit` / `exit`

**設計ポイント:**
- UseCase を `shared_ptr` で DI → プレゼンテーション層がインフラ層に依存しない
- 出力先を `std::ostream&` で受け取る → テスト可能な設計
- `tokenize()` で空白区切りの引数解析
- Result<T> の失敗をエラー表示してループ継続（落ちない設計）

**要手動確認（DevContainer 内）:**
- `make cli` でビルド確認
- `make run-cli` で動作確認（user create → user list → quit）
