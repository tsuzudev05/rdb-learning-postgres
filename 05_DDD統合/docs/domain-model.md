# ドメインモデル設計書

OKR管理ツールのDDD（ドメイン駆動設計）に基づく概念設計。  
クリーンアーキテクチャと組み合わせ、ドメイン知識をコードで表現することを目的とする。

---

## ユビキタス言語（Ubiquitous Language）

チーム全員が共通で使う用語を定義する。コード・ドキュメント・会話で統一して使用する。

| 用語（日本語） | 用語（英語） | 説明 |
|---|---|---|
| ユーザー | User | システムを利用するメンバー |
| チーム | Team | OKRを共有・管理するグループ |
| チームメンバー | TeamMember | チームに所属するUser。ロールを持つ |
| ロール | Role | admin（管理者）/ member（一般）|
| 期間 | Period | OKRを管理する半期単位のサイクル（H1/H2）|
| 目標 | Objective | 「何を達成したいか」を表す定性的な目標 |
| 主要な結果 | KeyResult | Objectiveの達成度を測る定量・定性指標 |
| 数値KR | NumericKeyResult | 数値で進捗を管理するKR |
| チェックKR | CheckboxKeyResult | 完了/未完了で管理するKR |
| 進捗ログ | ProgressLog | KRの進捗を時系列で記録したもの |
| オーナー | Owner | ObjectiveまたはKRの責任者となるUser |

---

## 集約設計（Aggregates）

DDDにおける集約境界と集約ルートを定義する。  
集約の外からは **必ず集約ルートを経由してアクセス** する。

```
┌─────────────────────────────────────────────────────┐
│  Team 集約                                          │
│  ┌─────────┐    ┌─────────────────┐                │
│  │  Team   │───▶│  TeamMember[]   │                │
│  │  (Root) │    │  - role         │                │
│  └─────────┘    └─────────────────┘                │
└─────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────┐
│  Period 集約                                        │
│  ┌──────────┐                                       │
│  │  Period  │                                       │
│  │  (Root)  │                                       │
│  └──────────┘                                       │
└─────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────┐
│  Objective 集約                                     │
│  ┌───────────┐                                      │
│  │ Objective │                                      │
│  │  (Root)   │                                      │
│  └───────────┘                                      │
└─────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────┐
│  KeyResult 集約                                     │
│  ┌───────────┐    ┌──────────────────────┐          │
│  │ KeyResult │───▶│  KrProgressLog[]     │          │
│  │  (Root)   │    │  - value / completed │          │
│  └───────────┘    └──────────────────────┘          │
└─────────────────────────────────────────────────────┘
```

### 集約分割の理由

`Objective` と `KeyResult` を別集約にした理由：
- KRの進捗更新（頻繁）とObjectiveの更新（稀）でトランザクション境界が異なる
- KRを独立して検索・更新するユースケースが存在する
- 集約を小さく保つことで、並行更新時のロック競合を減らす

---

## 値オブジェクト（Value Objects）

同一性ではなく **値の等価性** で比較されるオブジェクト。イミュータブルに設計する。

| 値オブジェクト | 説明 | 検証ルール |
|---|---|---|
| `UserId` | ユーザーID（UUID） | 有効なUUID形式 |
| `TeamId` | チームID（UUID） | 有効なUUID形式 |
| `PeriodId` | 期間ID（UUID） | 有効なUUID形式 |
| `ObjectiveId` | 目標ID（UUID） | 有効なUUID形式 |
| `KeyResultId` | KR ID（UUID） | 有効なUUID形式 |
| `Email` | メールアドレス | RFC5322形式 |
| `Role` | ロール | `admin` or `member` のみ |
| `Half` | 上期/下期 | `H1` or `H2` のみ |
| `NumericProgress` | 数値進捗 | target_value > 0、current_value >= 0 |
| `CheckboxProgress` | 完了進捗 | true / false |
| `KeyResultProgress` | KR進捗（Union型） | NumericProgress または CheckboxProgress |
| `DateRange` | 期間（開始〜終了） | start_date < end_date |

```cpp
// C++ 実装イメージ（値オブジェクト）
class Email {
public:
    static Result<Email> create(const std::string& value);
    const std::string& value() const { return value_; }
    bool operator==(const Email& other) const { return value_ == other.value_; }
private:
    explicit Email(std::string value) : value_(std::move(value)) {}
    std::string value_;
};

// KrProgressはVariantで型安全に表現
using KeyResultProgress = std::variant<NumericProgress, CheckboxProgress>;
```

---

## エンティティ（Entities）

同一性（ID）で識別されるオブジェクト。

| エンティティ | 集約 | 説明 |
|---|---|---|
| `User` | User集約ルート | ユーザー |
| `Team` | Team集約ルート | チーム |
| `TeamMember` | Team集約内 | チームメンバー（Team経由でのみ操作） |
| `Period` | Period集約ルート | 半期 |
| `Objective` | Objective集約ルート | 目標 |
| `KeyResult` | KeyResult集約ルート | 主要な結果 |
| `KrProgressLog` | KeyResult集約内 | 進捗ログ（KeyResult経由でのみ操作） |

---

## ドメインサービス（Domain Services）

複数の集約にまたがるビジネスルールを表現する。エンティティや値オブジェクトに収まらないロジックを置く。

### ObjectiveService

```
責務: Objectiveの作成・検証
ルール: owner_id が period の team のメンバーであること
```

```cpp
class ObjectiveService {
public:
    // owner が team のメンバーか検証してから Objective を生成する
    Result<Objective> createObjective(
        const TeamId& team_id,
        const PeriodId& period_id,
        const UserId& owner_id,
        const std::string& title
    );
};
```

### KeyResultService

```
責務: KRの進捗更新
ルール: 進捗ログを追加し、KR本体の current_value / is_completed を同期する
```

```cpp
class KeyResultService {
public:
    // ProgressLog を追加し、KeyResult の最新値を更新する
    Result<void> recordProgress(
        const KeyResultId& kr_id,
        const UserId& recorded_by,
        const KeyResultProgress& progress,
        const std::optional<std::string>& note
    );
};
```

---

## リポジトリインターフェース（Repository Interfaces）

ドメイン層で定義するインターフェース。実装はインフラ層（PostgreSQL）に置く。

```cpp
// ドメイン層（純粋仮想クラス）
class IUserRepository {
public:
    virtual ~IUserRepository() = default;
    virtual Result<User>              findById(const UserId& id) = 0;
    virtual Result<User>              findByEmail(const Email& email) = 0;
    virtual Result<void>              save(const User& user) = 0;
};

class IObjectiveRepository {
public:
    virtual ~IObjectiveRepository() = default;
    virtual Result<Objective>         findById(const ObjectiveId& id) = 0;
    virtual Result<std::vector<Objective>>
                                      findByPeriod(const PeriodId& period_id) = 0;
    virtual Result<void>              save(const Objective& objective) = 0;
};

class IKeyResultRepository {
public:
    virtual ~IKeyResultRepository() = default;
    virtual Result<KeyResult>         findById(const KeyResultId& id) = 0;
    virtual Result<std::vector<KeyResult>>
                                      findByObjective(const ObjectiveId& objective_id) = 0;
    virtual Result<void>              save(const KeyResult& kr) = 0;
    virtual Result<void>              saveProgressLog(const KrProgressLog& log) = 0;
};
```

---

## クリーンアーキテクチャ レイヤー構成

```
src/
├── domain/                 # ドメイン層（外部依存ゼロ）
│   ├── model/
│   │   ├── user/
│   │   │   ├── User.hpp
│   │   │   ├── UserId.hpp
│   │   │   └── Email.hpp
│   │   ├── team/
│   │   │   ├── Team.hpp
│   │   │   ├── TeamMember.hpp
│   │   │   └── Role.hpp
│   │   ├── period/
│   │   │   ├── Period.hpp
│   │   │   ├── Half.hpp
│   │   │   └── DateRange.hpp
│   │   ├── objective/
│   │   │   └── Objective.hpp
│   │   └── keyresult/
│   │       ├── KeyResult.hpp
│   │       ├── KeyResultProgress.hpp   ← std::variant
│   │       └── KrProgressLog.hpp
│   ├── service/
│   │   ├── ObjectiveService.hpp
│   │   └── KeyResultService.hpp
│   └── repository/                     # インターフェースのみ
│       ├── IUserRepository.hpp
│       ├── IObjectiveRepository.hpp
│       └── IKeyResultRepository.hpp
│
├── application/            # ユースケース層
│   └── usecase/
│       ├── CreateObjectiveUseCase.hpp
│       ├── UpdateKeyResultProgressUseCase.hpp
│       └── GetPeriodSummaryUseCase.hpp
│
├── infrastructure/         # インフラ層（PostgreSQL実装）
│   └── repository/
│       ├── PgUserRepository.cpp        ← libpqxx 実装
│       ├── PgObjectiveRepository.cpp
│       └── PgKeyResultRepository.cpp
│
└── presentation/           # プレゼンテーション層（CLI / API）
    └── cli/
        └── main.cpp
```

---

## 実装順序（推奨）

フェーズ2（ポートフォリオ制作）での実装順序。

```
1. domain/model の値オブジェクト（UserId, Email, Role, Half...）
2. domain/model のエンティティ（User, Team, Objective, KeyResult...）
3. domain/repository のインターフェース定義
4. infrastructure/repository の libpqxx 実装（Repositoryパターン）
5. domain/service のドメインサービス
6. application/usecase のユースケース
7. presentation/cli の CLI エントリポイント
```

---

## 関連ファイル

- [`../schema.sql`](../schema.sql) — データベーススキーマ
- [`../schema.md`](../schema.md) — スキーマ設計書
