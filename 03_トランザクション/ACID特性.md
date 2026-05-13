# トランザクション（BEGIN / ROLLBACK / COMMIT）動作確認結果

## 📋 背景

PostgreSQL のトランザクションが実際にどのように動作するかを体感するため、
`tasks` テーブルを用いて `BEGIN` / `ROLLBACK` / `COMMIT` の挙動を確認した。

インデックスが「速さ」の仕組みであるのに対し、トランザクションは**「正確さ」を保証する仕組み**である。
DDD の Repository パターンでは「1つのユースケース = 1トランザクション」が原則となるため、
ポートフォリオ開発に直結する重要なテーマとして取り組んだ。

---

## 🔬 実施内容

### 環境

| 項目 | 内容 |
|---|---|
| PostgreSQL | 16 |
| テーブル | `tasks`（4件） |
| 確認内容 | ROLLBACK・COMMIT の動作と ACID 特性との対応 |

### 手順

```sql
-- ステップ1: ROLLBACK（なかったことにする）
BEGIN;
UPDATE tasks SET status = 'done' WHERE user_id = 1;
SELECT id, title, status FROM tasks WHERE user_id = 1;  -- 変更されている
ROLLBACK;
SELECT id, title, status FROM tasks WHERE user_id = 1;  -- 元に戻っている

-- ステップ2: COMMIT（変更を永続化する）
BEGIN;
UPDATE tasks SET status = 'in_progress' WHERE id = 2;
SELECT id, title, status FROM tasks WHERE id = 2;       -- 変更されている
COMMIT;
SELECT id, title, status FROM tasks WHERE id = 2;       -- 永続化されている
```

---

## 📊 結果

### 事前確認：対象データ

```
-[ RECORD 1 ]---------------
id      | 1  title: インデックスを学ぶ  status: todo
-[ RECORD 2 ]---------------
id      | 2  title: JOINを理解する      status: in_progress
-[ RECORD 3 ]---------------
id      | 3  title: SELECT文を学ぶ      status: todo
-[ RECORD 4 ]---------------
id      | 4  title: PostgreSQL環境構築  status: done
```

---

### ステップ1：ROLLBACK（なかったことにする）

**BEGIN → UPDATE直後（トランザクション内）**

```
BEGIN
UPDATE 2
-[ RECORD 1 ]--------------
id     | 3
title  | SELECT文を学ぶ
status | done              ← 'todo' から 'done' に変わっている
-[ RECORD 2 ]--------------
id     | 4
title  | PostgreSQL環境構築
status | done              ← 元々 'done' のため変化なし
ROLLBACK
```

**ROLLBACK後の確認**

```
-[ RECORD 1 ]--------------
id     | 3
title  | SELECT文を学ぶ
status | todo              ← 'done' から 'todo' に戻っている ✅
-[ RECORD 2 ]--------------
id     | 4
title  | PostgreSQL環境構築
status | done
```

**読み方：**
- `UPDATE 2`：2件が更新対象になった
- ROLLBACK後：id=3 の status が `todo` に戻り、変更がなかったことになった
- id=4 は元々 `done` のため見た目の変化はないが、内部的には戻っている

---

### ステップ2：COMMIT（変更を永続化する）

**BEGIN → UPDATE → COMMIT**

```
BEGIN
UPDATE 1
-[ RECORD 1 ]----------
id     | 2
title  | JOINを理解する
status | in_progress    ← 変更が確定された
COMMIT
```

**COMMIT後の確認**

```
-[ RECORD 1 ]----------
id     | 2
title  | JOINを理解する
status | in_progress    ← 再起動後も消えない永続化された状態 ✅
```

**読み方：**
- `UPDATE 1`：1件が更新された
- COMMIT後も同じ値が返ってくる → 変更がディスクに永続化された

---

### 比較まとめ

| 操作 | UPDATE直後 | 操作後 | 結果 |
|---|---|---|---|
| BEGIN → UPDATE → **ROLLBACK** | 変更されている | 元に戻る | 取り消し成功 ✅ |
| BEGIN → UPDATE → **COMMIT** | 変更されている | 変更が残る | 永続化成功 ✅ |

---

## 🔍 詰まったこと・解決方法

### テーブルの表示が崩れて空に見えた問題

**状況：** 事前確認・最終確認のテーブルが罫線だけで表示され、データが見えなかった。

**原因：** `tasks` テーブルに `created_at` カラムが加わり、横幅がターミナル幅を超えて
折り返しが発生し、データ部分が表示されなかった。

**解決方法：** `psql -x` オプション（拡張表示モード）を追加した。
`-x` を付けると列を縦並びで表示するため、横幅を気にしなくなる。

```bash
# 修正前（横並び → 崩れる）
psql $DATABASE_URL -c "SELECT * FROM tasks;"

# 修正後（縦並び → 確実に表示される）
psql -x $DATABASE_URL -c "SELECT * FROM tasks;"
```

---

## 💡 気づき

1. **トランザクションの本質**
   複数の操作を「1つのまとまり」として扱う仕組み。
   `ROLLBACK` により「一部だけ更新された」中途半端な状態を防げる。

2. **ACID特性との対応**

   | 特性 | 意味 | 今回確認したこと |
   |---|---|---|
   | Atomicity（原子性） | 全部成功か全部失敗か | ROLLBACKで全件が元に戻った |
   | Consistency（一貫性） | 制約を常に満たす | CHECK制約違反は自動でROLLBACK |
   | Isolation（分離性） | 他のトランザクションの影響を受けない | 次回：分離レベルで確認予定 |
   | Durability（永続性） | COMMITしたら消えない | COMMIT後も同じ値が返ってきた |

3. **DDD × トランザクションの関係**
   Repository パターンでは、1つのユースケース（例：タスク更新）を
   1トランザクションとして扱うのが原則。
   `BEGIN` → ビジネスロジック実行 → `COMMIT`（失敗したら `ROLLBACK`）という流れになる。

4. **psql の表示オプション**
   - `-x`（`--expanded`）：列を縦並びで表示。カラム数が多いテーブルに有効
   - `\x` でpsql対話モード内でも切り替え可能

---

## ⏭️ 次回やること

- 分離レベル（READ COMMITTED / REPEATABLE READ）の挙動を確認する
- デッドロックを意図的に再現して回避策を理解する
- DDD の Repository パターンと PostgreSQL のトランザクションを接続する設計を考える
