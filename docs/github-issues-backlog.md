# GitHub Issues 登録候補（実装バックログ）

手動で Issues に登録するときの下書きリスト。  
登録後はこのファイルから削除して Issue 番号で管理する。

---

## 登録候補

### フェーズ6：アプリケーション層（ユースケース）実装
**タイトル案**: `feat: フェーズ6 ユースケース層の実装（CreateObjective / UpdateKrProgress）`
- `CreateObjectiveUseCase`：owner が team メンバーか検証してから保存
- `UpdateKeyResultProgressUseCase`：進捗ログ追加 + KR本体の現在値を同期
- トランザクション境界の設計（Unit of Work または Context 埋め込みパターン）
- **ラベル候補**: `enhancement`, `cpp`, `go`

### フェーズ7：ユニットテスト追加
**タイトル案**: `test: C++ / Go ドメイン層のユニットテストを追加`
- C++: Google Test を Makefile に追加
- Go: `domain/` 層は外部依存ゼロなので標準 testing パッケージで対応
- **ラベル候補**: `test`

### CI：GitHub Actions でビルド・テスト自動化
**タイトル案**: `ci: GitHub Actions でC++ビルドとGoテストを自動実行`
- C++: `make build` を Actions で実行
- Go: `go test ./...` を Actions で実行
- **ラベル候補**: `ci`, `devops`

### ポートフォリオ：Go 物流管理アプリの DBスキーマ設計
**タイトル案**: `feat: Go 物流管理アプリの DBスキーマ設計（ZOZOポートフォリオ向け）`
- **ラベル候補**: `enhancement`, `go`, `portfolio`

---

## 運用ルール

- Issue を作成したらこのリストから削除する
- コミットメッセージに `Closes #XX` を入れると PR マージ時に自動クローズ
- TASKS.md の「今週やること」には `（→ Issue #XX）` で参照するだけでよい
