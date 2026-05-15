# DBスキーマ設計書

OKR管理ツール（C++ × TypeScript × PostgreSQL）のデータベース設計。  
DDD / クリーンアーキテクチャに対応した設計とする。

---

## テーブル一覧

| テーブル名 | DDD位置付け | 概要 |
|---|---|---|
| `users` | 集約ルート: User | ユーザー |
| `teams` | 集約ルート: Team | チーム |
| `team_members` | Team集約内エンティティ | チームメンバー・ロール管理 |
| `periods` | 集約ルート: Period | 半期（OKRサイクル単位） |
| `objectives` | 集約ルート: Objective | 目標（OのO） |
| `key_results` | 集約ルート: KeyResult | 主要な結果（KR） |
| `kr_progress_logs` | KeyResult集約内エンティティ | 進捗履歴（時系列） |

---

## テーブル詳細

### users

| カラム | 型 | 制約 | 説明 |
|---|---|---|---|
| id | UUID | PK | ユーザーID |
| name | VARCHAR(100) | NOT NULL | 氏名 |
| email | VARCHAR(255) | NOT NULL, UNIQUE | メールアドレス |
| password_hash | VARCHAR(255) | NOT NULL | ハッシュ化済みパスワード |
| created_at | TIMESTAMPTZ | NOT NULL | 作成日時 |
| updated_at | TIMESTAMPTZ | NOT NULL | 更新日時 |

---

### teams

| カラム | 型 | 制約 | 説明 |
|---|---|---|---|
| id | UUID | PK | チームID |
| name | VARCHAR(100) | NOT NULL | チーム名 |
| description | TEXT | - | 説明 |
| created_at | TIMESTAMPTZ | NOT NULL | 作成日時 |
| updated_at | TIMESTAMPTZ | NOT NULL | 更新日時 |

---

### team_members

チームとユーザーの中間テーブル。`role` により管理者とメンバーを区別する。

| カラム | 型 | 制約 | 説明 |
|---|---|---|---|
| id | UUID | PK | - |
| team_id | UUID | FK → teams | チームID |
| user_id | UUID | FK → users | ユーザーID |
| role | VARCHAR(20) | CHECK: admin\|member | ロール |
| joined_at | TIMESTAMPTZ | NOT NULL | 参加日時 |

**UNIQUE制約**：`(team_id, user_id)`

---

### periods

半期単位のOKRサイクルを管理する。

| カラム | 型 | 制約 | 説明 |
|---|---|---|---|
| id | UUID | PK | 期間ID |
| team_id | UUID | FK → teams | 対象チーム |
| name | VARCHAR(50) | NOT NULL | 例: `2025-H1` |
| half | VARCHAR(2) | CHECK: H1\|H2 | 上期 / 下期 |
| start_date | DATE | NOT NULL | 開始日 |
| end_date | DATE | NOT NULL | 終了日 |
| created_at | TIMESTAMPTZ | NOT NULL | 作成日時 |
| updated_at | TIMESTAMPTZ | NOT NULL | 更新日時 |

**CHECK制約**：`start_date < end_date`

---

### objectives

OKRの「O（目標）」。`owner_id` が対象チームのメンバーかどうかは **ドメインサービス層で検証** する。

| カラム | 型 | 制約 | 説明 |
|---|---|---|---|
| id | UUID | PK | 目標ID |
| period_id | UUID | FK → periods | 対象期間 |
| owner_id | UUID | FK → users | 目標オーナー |
| title | VARCHAR(255) | NOT NULL | 目標タイトル |
| description | TEXT | - | 詳細説明 |
| display_order | INT | NOT NULL | UI表示順 |
| created_at | TIMESTAMPTZ | NOT NULL | 作成日時 |
| updated_at | TIMESTAMPTZ | NOT NULL | 更新日時 |

---

### key_results

OKRの「KR（主要な結果）」。`kr_type` により数値管理とチェックボックス管理を切り替える。

| カラム | 型 | 制約 | 説明 |
|---|---|---|---|
| id | UUID | PK | KR ID |
| objective_id | UUID | FK → objectives | 親目標 |
| owner_id | UUID | FK → users | KRオーナー |
| title | VARCHAR(255) | NOT NULL | KRタイトル |
| kr_type | VARCHAR(20) | CHECK: numeric\|checkbox | 進捗種別 |
| target_value | NUMERIC | nullable | 目標値（numeric のみ） |
| current_value | NUMERIC | nullable | 現在値（numeric のみ） |
| is_completed | BOOLEAN | NOT NULL | 完了フラグ（checkbox のみ） |
| display_order | INT | NOT NULL | UI表示順 |
| created_at | TIMESTAMPTZ | NOT NULL | 作成日時 |
| updated_at | TIMESTAMPTZ | NOT NULL | 更新日時 |

**CHECK制約**：`kr_type = 'numeric'` の場合 `target_value IS NOT NULL`

---

### kr_progress_logs

KRの進捗を時系列で記録する。KR本体は最新値のみ保持し、履歴はこのテーブルに積む。

| カラム | 型 | 制約 | 説明 |
|---|---|---|---|
| id | UUID | PK | ログID |
| key_result_id | UUID | FK → key_results | 対象KR |
| recorded_by | UUID | FK → users | 記録者 |
| value | NUMERIC | nullable | 数値進捗（numeric のみ） |
| completed | BOOLEAN | nullable | 完了フラグ（checkbox のみ） |
| note | TEXT | - | コメント |
| recorded_at | TIMESTAMPTZ | NOT NULL | 記録日時 |

---

## インデックス一覧

| インデックス名 | 対象 | 目的 |
|---|---|---|
| idx_team_members_team_id | team_members(team_id) | チーム別メンバー検索 |
| idx_team_members_user_id | team_members(user_id) | ユーザー別チーム検索 |
| idx_periods_team_id | periods(team_id) | チーム別期間検索 |
| idx_objectives_period_id | objectives(period_id) | 期間別目標検索 |
| idx_objectives_owner_id | objectives(owner_id) | オーナー別目標検索 |
| idx_key_results_objective_id | key_results(objective_id) | 目標別KR検索 |
| idx_kr_progress_logs_kr_id | kr_progress_logs(key_result_id) | KR別履歴検索 |
| idx_kr_progress_logs_recorded | kr_progress_logs(recorded_at) | 時系列検索 |

---

## 設計上の注意点

### kr_type によるドメイン分岐

`kr_type` の `numeric` / `checkbox` の切り替えはアプリケーション層・ドメイン層で処理する。  
DBレベルではCHECK制約と `nullable` で整合性を保ち、C++のドメイン層では `std::variant` を用いた値オブジェクトで表現する。

```cpp
// ドメイン層での表現例（C++）
class KeyResultProgress {
  std::variant<NumericProgress, CheckboxProgress> value_;
};
```

### owner_id の整合性はドメインサービスが保証

`objectives.owner_id` と `key_results.owner_id` が対象チームのメンバーであるかどうかは、  
外部キー制約では表現できないため **ドメインサービス層で検証** する。

```cpp
// ドメインサービス例（C++）
class ObjectiveService {
public:
  Result<Objective> create(TeamId team_id, UserId owner_id, PeriodId period_id, ...);
  // owner_id が team_id のメンバーか検証してから Repository に渡す
};
```

### updated_at の自動更新

PostgreSQLトリガー `update_updated_at()` により、更新時に自動セットされる。  
アプリケーション層での手動セットは不要。

---

## 関連ファイル

- [`schema.sql`](./schema.sql) — CREATE TABLE 本体
- [`docs/domain-model.md`](./docs/domain-model.md) — ドメインモデル・集約設計
