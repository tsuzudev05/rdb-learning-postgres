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
│   ├── service/                    # ドメインサービス
│   └── repository/                 # Repositoryインターフェース（純粋仮想）
├── application/usecase/            # ユースケース層
├── infrastructure/repository/      # インフラ層（libpqxx 実装）
└── presentation/cli/               # プレゼンテーション層
```

**レイヤールール:** `domain/` は STL のみ。`infrastructure/` のみ libpqxx に依存。上位→下位の依存のみ許可。

---

## 実装方針

### エラーハンドリング

`std::expected`（C++23）の自前実装 `Result<T>` を使用。例外は使わない。

```cpp
Result<UserId>::ok(UserId{value});
Result<UserId>::err("UserId: 無効なUUID形式です: " + value);
auto result = UserId::create("...");
if (!result) { /* result.error() でメッセージ取得 */ }
```

### 値オブジェクト設計方針

- イミュータブル（private コンストラクタ + ファクトリパターン）
- バリデーションは `create()` で行い `Result<T>` を返す
- 等価比較演算子（`==` / `!=`）を必ず実装する

---

## 実装状況（05_DDD統合/）

| フェーズ | 内容 | 状態 |
| -------- | ---- | ---- |
| フェーズ1 | 値オブジェクト（UserId / Email / Role / Half / DateRange / KeyResultProgress） | ✅ |
| フェーズ2 | エンティティ（User / Team / TeamMember / Period / Objective / KeyResult / KrProgressLog） | ✅ |
| フェーズ3 | Repository インターフェース（I*Repository.hpp × 5） | ✅ |
| フェーズ4 | libpqxx 実装（Pg*Repository.hpp × 5、upsert / 集約一括管理） | ✅ |
| フェーズ5 | Go (pgx) 実装：User / Team / Period / Objective / KeyResult（go/domain・infrastructure・cmd/smoke） | ✅ |
| フェーズ6 | C++ ユースケース層（*UseCase.hpp × 5、DI・Result<T>・DTO 分離） | ✅ |
| フェーズ7 | C++ CLI プレゼンテーション層（CliApp.hpp・main_cli.cpp・Makefile） | ✅ |
| フェーズ8-1 | Go echo セットアップ・ヘルスチェック API（cmd/api/main.go・Makefile） | ✅ |
| フェーズ8-2 | Go User / Team API エンドポイント（handler/user_handler.go・team_handler.go） | ✅ |
| フェーズ8-3 | Go Period / Objective API エンドポイント（handler/period_handler.go・objective_handler.go） | ✅ |
| フェーズ8-4 | Go KeyResult API エンドポイント（handler/keyresult_handler.go）・統一エラーハンドリング | ✅ |

### Go API エンドポイント一覧

```
GET/POST   /api/v1/users
GET/DELETE /api/v1/users/:id
GET/POST   /api/v1/teams
GET/DELETE /api/v1/teams/:id
POST/DELETE /api/v1/teams/:id/members[/:user_id]
GET/POST   /api/v1/periods
GET/DELETE /api/v1/periods/:id
GET/POST   /api/v1/objectives
GET/DELETE /api/v1/objectives/:id
GET/POST   /api/v1/key_results          # ?objective_id= or ?owner_id=
GET/DELETE /api/v1/key_results/:id
```

---

## DBスキーマ

- ファイル: `05_DDD統合/schema.sql` / `schema.md`
- ドメインモデル: `05_DDD統合/docs/domain-model.md`
- PostgreSQL 16 / uuid-ossp 拡張 / `updated_at` トリガー自動更新

| テーブル | 集約 |
| -------- | ---- |
| `users` | User |
| `teams` + `team_members` | Team |
| `periods` | Period |
| `objectives` | Objective |
| `key_results` + `kr_progress_logs` | KeyResult |

---

## ユビキタス言語

| 日本語 | 英語 | 説明 |
| ------ | ---- | ---- |
| ユーザー | User | システム利用者 |
| チーム | Team | OKRを共有するグループ |
| チームメンバー | TeamMember | チームに所属するUser |
| ロール | Role | admin / member |
| 期間 | Period | 半期単位のOKRサイクル |
| 目標 | Objective | 定性的な目標（OのO） |
| 主要な結果 | KeyResult | 達成度を測る指標（KR） |
| 進捗ログ | ProgressLog | KRの時系列履歴 |
| オーナー | Owner | ObjectiveまたはKRの責任者 |

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
