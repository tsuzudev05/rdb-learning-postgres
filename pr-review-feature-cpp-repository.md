# PR コードレビュー
**対象ブランチ**: `feature/cpp-repository` → `main`  
**レビュー日**: 2026-05-20  
**変更規模**: 70 ファイル / +4,307 行

---

## 📝 概要

OKR管理ツールの DDD ドメイン層〜インフラ層を段階的に実装した PR。
C++17 + libpqxx と Go + pgx v5 の 2 言語で同じ設計を並走させており、学習目的として非常に明確な構造になっている。

---

## ✅ 良い点

### DDD 設計

- **`UuidId<Tag>` テンプレート**が秀逸。`UserId` と `TeamId` を型で区別できるため、混同を**コンパイル時に**防げる
- **`std::variant<NumericProgress, CheckboxProgress>`** による Union 型の値オブジェクトは、Go の `interface` よりも型安全で、`std::visit` で網羅性も保証できる
- **集約ルールの実装が正確**（`Team::removeMember` の「admin が 0 人になる場合はエラー」など、ドメインロジックがエンティティ内に収まっている）
- **インターフェースの分離**が徹底されており、`domain/` 層は外部ライブラリへの依存がゼロ

### C++ 実装

- `Result<T>` / `Result<void>` の自前実装が簡潔でわかりやすい
- `PgKeyResultRepository` の JOIN 結果から集約を再構築するロジック（`unordered_map` + `vector` によるグルーピング）が正確
- `KrProgressLog` の `ON CONFLICT DO NOTHING`（append-only）が、不変の履歴という設計意図をよく表現している
- upsert（`ON CONFLICT (id) DO UPDATE`）で save() の新規・更新を統一している点が Repository インターフェースの意図に合致

### Go 実装

- `var _ domainrepo.UserRepository = (*PgUserRepository)(nil)` によるコンパイル時チェックが入っている
- `pgxpool.Pool` 採用により goroutine セーフ・並行接続対応になっている
- `scanner` インターフェースで `pgx.Row` / `pgx.Rows` の両方を抽象化しており、`scanUser()` のコード重複がない
- `context.Context` を全メソッドに伝播させており、キャンセル・タイムアウトに対応できる

---

## 🔴 要修正

### 1. `setup.sh` — `DROP SCHEMA CASCADE` が毎回実行される（高リスク）

```bash
# setup.sh (現状)
psql ... -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public; ..."
```

**問題**: DevContainer を再起動するたびに全データが消える。意図した動作であれば問題ないが、将来的に永続データを扱うようになったとき危険。

**提案**:

```bash
# テーブルが存在しない場合のみ投入する
TABLE_EXISTS=$(psql ... -tAc "SELECT to_regclass('public.users')")
if [ -z "$TABLE_EXISTS" ]; then
  psql ... -f /workspace/05_DDD統合/schema.sql
  echo "✅ DDD スキーマ投入完了"
else
  echo "⏭️  スキーマ投入済み（スキップ）"
fi
```

---

### 2. `setup.sh` — メールアドレスのハードコード

```bash
git config --global user.email "tsuzu.develop.05@gmail.com"
```

**問題**: 個人のメールアドレスがリポジトリに残り、他者がフォーク・利用する際に誤ったコミット情報が設定される。

**提案**: `.env` ファイルや `devcontainer.json` の `remoteEnv` で外部から注入する（削除した元の実装に近い形に戻す）。

---

## 🟡 改善提案（次のステップ向け）

### 3. C++ エンティティ — タイムスタンプの `setTimestamps()` ミューテーション

```cpp
// Period.hpp など
void setTimestamps(std::string createdAt, std::string updatedAt) {
    createdAt_ = std::move(createdAt);  // ← 構築後のミューテーション
    updatedAt_ = std::move(updatedAt);
}
```

**問題**: DB 再構築用のファクトリ経由で設定できれば、エンティティのイミュータビリティを守れる。

**提案**: DB 再構築専用の `reconstruct()` static メソッドを用意し、タイムスタンプも含めてまとめて初期化する：

```cpp
static Result<Period> reconstruct(
    PeriodId id, TeamId teamId, std::string name,
    Half half, DateRange dateRange,
    std::string createdAt, std::string updatedAt
);
```

---

### 4. C++ — タイムスタンプが `std::string` で保持されている

```cpp
std::string createdAt_;   // "2026-05-19 12:34:56.000000+00"
std::string updatedAt_;
```

**問題**: ソート・比較・差分計算などが文字列操作になる。DB 再構築時は `std::string` が手軽だが、将来の活用を考えると `std::chrono::system_clock::time_point` の方が適切。

**提案**: ユースケース層を実装するタイミングで `std::chrono` に切り替えることを検討する。

---

### 5. Go — `errors.go` の `errorf` ヘルパーが不要

```go
// errors.go
func errorf(format string, args ...any) error {
    return fmt.Errorf(format, args...)
}
```

`user.go` 内で `fmt.Errorf` を直接呼べば十分。このファイルは削除してよい。

---

### 6. トランザクション境界の設計（将来課題）

現状の Go 実装では、Repository の各メソッドが独立してクエリを実行する。ユースケース層を実装する際、**複数 Repository にまたがる操作をアトミックにする仕組み**が必要になる。

**提案**: 以下のいずれかのパターンを検討する：

```go
// パターンA: Context にトランザクションを埋め込む
type txKey struct{}
func TxFromContext(ctx context.Context) pgx.Tx { ... }

// パターンB: Unit of Work パターンで複数 Repository をまとめる
type UnitOfWork interface {
    UserRepository() UserRepository
    Commit() error
    Rollback() error
}
```

---

### 7. テストカバレッジ（将来課題）

現状はスモークテスト（統合テスト）のみ。ドメイン層（値オブジェクト・エンティティ）は **外部依存がゼロ**なので、ユニットテストを追加しやすい。

```go
// 例: user_id_test.go
func TestNewUserId_invalid(t *testing.T) {
    _, err := user.NewUserId("not-a-uuid")
    assert.Error(t, err)
}
```

C++ 側は Google Test などを Makefile に追加することで対応可能。

---

## 📊 総評

| 観点 | 評価 | コメント |
|---|---|---|
| DDD 設計 | ⭐⭐⭐⭐⭐ | 集約・値オブジェクト・Repository 分離が正確 |
| C++ 実装品質 | ⭐⭐⭐⭐ | `Result<T>` / `std::variant` の活用が適切。タイムスタンプ設計に改善余地あり |
| Go 実装品質 | ⭐⭐⭐⭐ | 慣用的な Go のパターンに準拠。`errors.go` は不要 |
| DevContainer 設定 | ⭐⭐⭐ | DROP CASCADE が毎回実行される点を修正推奨 |
| テスト | ⭐⭐⭐ | スモークテストで動作確認済み。ユニットテスト追加を推奨 |
| ドキュメント | ⭐⭐⭐⭐⭐ | `CLAUDE.md` / `domain-model.md` / `schema.md` が充実 |

**マージ判断**: 要修正の #1（DROP CASCADE）と #2（メールのハードコード）を対応すればマージ可。
それ以外の改善点はユースケース層実装時の次 PR で対応する形でよい。
