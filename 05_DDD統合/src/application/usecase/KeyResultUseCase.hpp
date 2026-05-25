#pragma once

#include <memory>
#include <string>
#include <optional>
#include <vector>
#include "../../domain/model/keyresult/KeyResult.hpp"
#include "../../domain/model/keyresult/KeyResultId.hpp"
#include "../../domain/model/keyresult/KeyResultProgress.hpp"
#include "../../domain/model/keyresult/KrProgressLog.hpp"
#include "../../domain/model/keyresult/KrProgressLogId.hpp"
#include "../../domain/model/objective/ObjectiveId.hpp"
#include "../../domain/model/user/UserId.hpp"
#include "../../domain/repository/IKeyResultRepository.hpp"
#include "../../common/Result.hpp"

// =============================================================================
// KeyResultUseCase — 主要な結果ユースケース（アプリケーション層）
//
// ドメイン層の IKeyResultRepository に依存注入（DI）パターンで依存する。
// インフラ層（libpqxx）への直接依存を持たない。
//
// 提供するユースケース:
//   - CreateKeyResult          : 新規 KR を作成する
//   - GetKeyResult             : ID で KR を1件取得する（進捗ログ含む）
//   - ListKeyResultsByObjective: 目標IDに紐づく KR 一覧を取得する（display_order 昇順）
//   - ListKeyResultsByOwner    : オーナーIDに紐づく KR 一覧を取得する
//   - UpdateProgress           : KR の進捗を更新し、進捗ログを追記する
//   - DeleteKeyResult          : ID で KR を削除する
//
// 進捗種別（kr_type）:
//   - "numeric"  : 数値目標（例: 売上 100万円）→ NumericProgress
//   - "checkbox" : 完了フラグ（例: 設計書を完成させる）→ CheckboxProgress
//
// エラーハンドリング:
//   すべての操作は Result<T> を返す。例外は使わない。
// =============================================================================

// ─── 入力 DTO ──────────────────────────────────────────────────────────────

// numeric KR 作成用
struct CreateNumericKeyResultInput {
    std::string objectiveId;  // UUID 文字列
    std::string ownerId;      // UUID 文字列
    std::string title;
    double targetValue;       // 目標値（> 0）
    double currentValue = 0.0;// 現在値（>= 0、省略時は 0）
    int displayOrder = 0;
};

// checkbox KR 作成用
struct CreateCheckboxKeyResultInput {
    std::string objectiveId;  // UUID 文字列
    std::string ownerId;      // UUID 文字列
    std::string title;
    bool completed = false;   // 初期状態（省略時は未完了）
    int displayOrder = 0;
};

struct GetKeyResultInput {
    std::string keyResultId;  // UUID 文字列
};

struct DeleteKeyResultInput {
    std::string keyResultId;  // UUID 文字列
};

struct ListKeyResultsByObjectiveInput {
    std::string objectiveId;  // UUID 文字列
};

struct ListKeyResultsByOwnerInput {
    std::string ownerId;      // UUID 文字列
};

// numeric KR の進捗更新用
struct UpdateNumericProgressInput {
    std::string keyResultId;  // UUID 文字列
    std::string recordedBy;   // UUID 文字列（記録者）
    double newValue;          // 新しい現在値（>= 0）
    std::optional<std::string> note;  // メモ（省略可）
};

// checkbox KR の進捗更新用
struct UpdateCheckboxProgressInput {
    std::string keyResultId;  // UUID 文字列
    std::string recordedBy;   // UUID 文字列（記録者）
    bool completed;           // 新しい完了フラグ
    std::optional<std::string> note;  // メモ（省略可）
};

// ─── 出力 DTO ──────────────────────────────────────────────────────────────

struct KrProgressLogOutput {
    std::string id;
    std::string keyResultId;
    std::string recordedBy;
    std::optional<double> value;      // numeric KR の記録値
    std::optional<bool> completed;    // checkbox KR の記録値
    std::optional<std::string> note;
    std::string recordedAt;

    static KrProgressLogOutput from(const KrProgressLog& log) {
        return KrProgressLogOutput{
            log.id().value(),
            log.keyResultId().value(),
            log.recordedBy().value(),
            log.value(),
            log.completed(),
            log.note(),
            log.recordedAt()
        };
    }
};

struct KeyResultOutput {
    std::string id;
    std::string objectiveId;
    std::string ownerId;
    std::string title;
    std::string krType;           // "numeric" or "checkbox"
    std::optional<double> targetValue;   // numeric のみ
    std::optional<double> currentValue;  // numeric のみ
    std::optional<bool> completed;       // checkbox のみ
    double achievementRate;       // 0.0〜1.0
    int displayOrder;
    std::vector<KrProgressLogOutput> progressLogs;
    std::string createdAt;
    std::string updatedAt;

    // KeyResult エンティティから変換するファクトリ
    static KeyResultOutput from(const KeyResult& kr) {
        KeyResultOutput out;
        out.id            = kr.id().value();
        out.objectiveId   = kr.objectiveId().value();
        out.ownerId       = kr.ownerId().value();
        out.title         = kr.title();
        out.krType        = kr.krType();
        out.achievementRate = kr.achievementRate();
        out.displayOrder  = kr.displayOrder();
        out.createdAt     = kr.createdAt();
        out.updatedAt     = kr.updatedAt();

        // 進捗値を種別ごとに展開
        if (std::holds_alternative<NumericProgress>(kr.progress())) {
            const auto& np = std::get<NumericProgress>(kr.progress());
            out.targetValue  = np.targetValue();
            out.currentValue = np.currentValue();
        } else {
            const auto& cp = std::get<CheckboxProgress>(kr.progress());
            out.completed = cp.isCompleted();
        }

        // 進捗ログ
        out.progressLogs.reserve(kr.progressLogs().size());
        for (const auto& log : kr.progressLogs()) {
            out.progressLogs.push_back(KrProgressLogOutput::from(log));
        }

        return out;
    }
};

// ─── KeyResultUseCase ───────────────────────────────────────────────────────

class KeyResultUseCase {
public:
    // コンストラクタ：IKeyResultRepository を DI で受け取る（所有権は共有しない）
    explicit KeyResultUseCase(std::shared_ptr<IKeyResultRepository> repo)
        : repo_(std::move(repo)) {}

    // ──────────────────────────────────────────────────────────────────────
    // CreateNumericKeyResult — 数値目標 KR を新規作成する
    //
    // 処理フロー:
    //   1. 入力値を各値オブジェクトに変換（バリデーション含む）
    //   2. KeyResultId を新規生成
    //   3. KeyResult エンティティを生成
    //   4. Repository#save で永続化
    // ──────────────────────────────────────────────────────────────────────
    Result<KeyResultOutput> createNumericKeyResult(const CreateNumericKeyResultInput& input) {
        // 1. ObjectiveId を生成（UUID バリデーション）
        auto objIdResult = ObjectiveId::create(input.objectiveId);
        if (!objIdResult) {
            return Result<KeyResultOutput>::err(objIdResult.error());
        }

        // 2. UserId（ownerId）を生成（UUID バリデーション）
        auto ownerIdResult = UserId::create(input.ownerId);
        if (!ownerIdResult) {
            return Result<KeyResultOutput>::err(ownerIdResult.error());
        }

        // 3. NumericProgress を生成（targetValue > 0, currentValue >= 0 バリデーション）
        auto progressResult = NumericProgress::create(input.targetValue, input.currentValue);
        if (!progressResult) {
            return Result<KeyResultOutput>::err(progressResult.error());
        }

        // 4. KeyResultId を新規生成
        auto idResult = KeyResultId::generate();
        if (!idResult) {
            return Result<KeyResultOutput>::err(idResult.error());
        }

        // 5. KeyResult エンティティを生成（title・displayOrder バリデーション含む）
        KeyResultProgress progress = progressResult.value();
        auto krResult = KeyResult::create(
            idResult.value(),
            objIdResult.value(),
            ownerIdResult.value(),
            input.title,
            progress,
            input.displayOrder
        );
        if (!krResult) {
            return Result<KeyResultOutput>::err(krResult.error());
        }

        // 6. 永続化
        auto saveResult = repo_->save(krResult.value());
        if (!saveResult) {
            return Result<KeyResultOutput>::err(saveResult.error());
        }

        return Result<KeyResultOutput>::ok(KeyResultOutput::from(krResult.value()));
    }

    // ──────────────────────────────────────────────────────────────────────
    // CreateCheckboxKeyResult — 完了フラグ KR を新規作成する
    // ──────────────────────────────────────────────────────────────────────
    Result<KeyResultOutput> createCheckboxKeyResult(const CreateCheckboxKeyResultInput& input) {
        // 1. ObjectiveId を生成（UUID バリデーション）
        auto objIdResult = ObjectiveId::create(input.objectiveId);
        if (!objIdResult) {
            return Result<KeyResultOutput>::err(objIdResult.error());
        }

        // 2. UserId（ownerId）を生成（UUID バリデーション）
        auto ownerIdResult = UserId::create(input.ownerId);
        if (!ownerIdResult) {
            return Result<KeyResultOutput>::err(ownerIdResult.error());
        }

        // 3. CheckboxProgress を生成
        CheckboxProgress cp = CheckboxProgress::create(input.completed);

        // 4. KeyResultId を新規生成
        auto idResult = KeyResultId::generate();
        if (!idResult) {
            return Result<KeyResultOutput>::err(idResult.error());
        }

        // 5. KeyResult エンティティを生成
        KeyResultProgress progress = cp;
        auto krResult = KeyResult::create(
            idResult.value(),
            objIdResult.value(),
            ownerIdResult.value(),
            input.title,
            progress,
            input.displayOrder
        );
        if (!krResult) {
            return Result<KeyResultOutput>::err(krResult.error());
        }

        // 6. 永続化
        auto saveResult = repo_->save(krResult.value());
        if (!saveResult) {
            return Result<KeyResultOutput>::err(saveResult.error());
        }

        return Result<KeyResultOutput>::ok(KeyResultOutput::from(krResult.value()));
    }

    // ──────────────────────────────────────────────────────────────────────
    // GetKeyResult — ID で KR を1件取得する（進捗ログ含む）
    //
    // 対象が存在しない場合はエラー（NotFound）を返す。
    // ──────────────────────────────────────────────────────────────────────
    Result<KeyResultOutput> getKeyResult(const GetKeyResultInput& input) {
        // KeyResultId を生成（UUID バリデーション）
        auto idResult = KeyResultId::create(input.keyResultId);
        if (!idResult) {
            return Result<KeyResultOutput>::err(idResult.error());
        }

        // Repository から取得
        auto findResult = repo_->findById(idResult.value());
        if (!findResult) {
            return Result<KeyResultOutput>::err(findResult.error());
        }
        if (!findResult.value().has_value()) {
            return Result<KeyResultOutput>::err(
                "GetKeyResult: KR が見つかりません: " + input.keyResultId
            );
        }

        return Result<KeyResultOutput>::ok(KeyResultOutput::from(findResult.value().value()));
    }

    // ──────────────────────────────────────────────────────────────────────
    // ListKeyResultsByObjective — 目標IDに紐づく KR 一覧を取得する（display_order 昇順）
    // ──────────────────────────────────────────────────────────────────────
    Result<std::vector<KeyResultOutput>> listKeyResultsByObjective(
        const ListKeyResultsByObjectiveInput& input
    ) {
        // ObjectiveId を生成（UUID バリデーション）
        auto idResult = ObjectiveId::create(input.objectiveId);
        if (!idResult) {
            return Result<std::vector<KeyResultOutput>>::err(idResult.error());
        }

        auto findResult = repo_->findByObjectiveId(idResult.value());
        if (!findResult) {
            return Result<std::vector<KeyResultOutput>>::err(findResult.error());
        }

        std::vector<KeyResultOutput> outputs;
        outputs.reserve(findResult.value().size());
        for (const auto& kr : findResult.value()) {
            outputs.push_back(KeyResultOutput::from(kr));
        }

        return Result<std::vector<KeyResultOutput>>::ok(std::move(outputs));
    }

    // ──────────────────────────────────────────────────────────────────────
    // ListKeyResultsByOwner — オーナーIDに紐づく KR 一覧を取得する
    // ──────────────────────────────────────────────────────────────────────
    Result<std::vector<KeyResultOutput>> listKeyResultsByOwner(
        const ListKeyResultsByOwnerInput& input
    ) {
        // UserId（ownerId）を生成（UUID バリデーション）
        auto idResult = UserId::create(input.ownerId);
        if (!idResult) {
            return Result<std::vector<KeyResultOutput>>::err(idResult.error());
        }

        auto findResult = repo_->findByOwnerId(idResult.value());
        if (!findResult) {
            return Result<std::vector<KeyResultOutput>>::err(findResult.error());
        }

        std::vector<KeyResultOutput> outputs;
        outputs.reserve(findResult.value().size());
        for (const auto& kr : findResult.value()) {
            outputs.push_back(KeyResultOutput::from(kr));
        }

        return Result<std::vector<KeyResultOutput>>::ok(std::move(outputs));
    }

    // ──────────────────────────────────────────────────────────────────────
    // UpdateNumericProgress — 数値 KR の進捗を更新し、進捗ログを追記する
    //
    // 処理フロー:
    //   1. KR を取得（存在チェック・numeric 種別チェック）
    //   2. NumericProgress を新規生成（バリデーション）
    //   3. KrProgressLogId を新規生成
    //   4. KrProgressLog エンティティを生成
    //   5. KeyResult.updateProgress()（進捗種別一致チェック付き）
    //   6. Repository#save で永続化
    // ──────────────────────────────────────────────────────────────────────
    Result<KeyResultOutput> updateNumericProgress(const UpdateNumericProgressInput& input) {
        // 1. KeyResultId を生成（UUID バリデーション）
        auto krIdResult = KeyResultId::create(input.keyResultId);
        if (!krIdResult) {
            return Result<KeyResultOutput>::err(krIdResult.error());
        }

        // 2. UserId（recordedBy）を生成（UUID バリデーション）
        auto recorderIdResult = UserId::create(input.recordedBy);
        if (!recorderIdResult) {
            return Result<KeyResultOutput>::err(recorderIdResult.error());
        }

        // 3. KR を取得（存在チェック）
        auto findResult = repo_->findById(krIdResult.value());
        if (!findResult) {
            return Result<KeyResultOutput>::err(findResult.error());
        }
        if (!findResult.value().has_value()) {
            return Result<KeyResultOutput>::err(
                "UpdateNumericProgress: KR が見つかりません: " + input.keyResultId
            );
        }
        KeyResult kr = std::move(findResult.value().value());

        // 4. numeric KR であることを確認
        if (kr.krType() != "numeric") {
            return Result<KeyResultOutput>::err(
                "UpdateNumericProgress: numeric KR ではありません（種別: " + kr.krType() + "）"
            );
        }

        // 5. targetValue は現在の値を継承し、currentValue のみ更新
        const auto& currentProgress = std::get<NumericProgress>(kr.progress());
        auto newProgressResult = NumericProgress::create(
            currentProgress.targetValue(),
            input.newValue
        );
        if (!newProgressResult) {
            return Result<KeyResultOutput>::err(newProgressResult.error());
        }

        // 6. KrProgressLogId を新規生成
        auto logIdResult = KrProgressLogId::generate();
        if (!logIdResult) {
            return Result<KeyResultOutput>::err(logIdResult.error());
        }

        // 7. KrProgressLog を生成（numeric 用）
        auto logResult = KrProgressLog::createNumeric(
            logIdResult.value(),
            krIdResult.value(),
            recorderIdResult.value(),
            input.newValue,
            input.note
        );
        if (!logResult) {
            return Result<KeyResultOutput>::err(logResult.error());
        }

        // 8. ドメイン操作：進捗更新（進捗種別一致チェック付き）
        KeyResultProgress newProgress = newProgressResult.value();
        auto updateResult = kr.updateProgress(std::move(newProgress), std::move(logResult.value()));
        if (!updateResult) {
            return Result<KeyResultOutput>::err(updateResult.error());
        }

        // 9. 永続化
        auto saveResult = repo_->save(kr);
        if (!saveResult) {
            return Result<KeyResultOutput>::err(saveResult.error());
        }

        return Result<KeyResultOutput>::ok(KeyResultOutput::from(kr));
    }

    // ──────────────────────────────────────────────────────────────────────
    // UpdateCheckboxProgress — checkbox KR の進捗を更新し、進捗ログを追記する
    // ──────────────────────────────────────────────────────────────────────
    Result<KeyResultOutput> updateCheckboxProgress(const UpdateCheckboxProgressInput& input) {
        // 1. KeyResultId を生成（UUID バリデーション）
        auto krIdResult = KeyResultId::create(input.keyResultId);
        if (!krIdResult) {
            return Result<KeyResultOutput>::err(krIdResult.error());
        }

        // 2. UserId（recordedBy）を生成（UUID バリデーション）
        auto recorderIdResult = UserId::create(input.recordedBy);
        if (!recorderIdResult) {
            return Result<KeyResultOutput>::err(recorderIdResult.error());
        }

        // 3. KR を取得（存在チェック）
        auto findResult = repo_->findById(krIdResult.value());
        if (!findResult) {
            return Result<KeyResultOutput>::err(findResult.error());
        }
        if (!findResult.value().has_value()) {
            return Result<KeyResultOutput>::err(
                "UpdateCheckboxProgress: KR が見つかりません: " + input.keyResultId
            );
        }
        KeyResult kr = std::move(findResult.value().value());

        // 4. checkbox KR であることを確認
        if (kr.krType() != "checkbox") {
            return Result<KeyResultOutput>::err(
                "UpdateCheckboxProgress: checkbox KR ではありません（種別: " + kr.krType() + "）"
            );
        }

        // 5. CheckboxProgress を生成
        CheckboxProgress cp = CheckboxProgress::create(input.completed);

        // 6. KrProgressLogId を新規生成
        auto logIdResult = KrProgressLogId::generate();
        if (!logIdResult) {
            return Result<KeyResultOutput>::err(logIdResult.error());
        }

        // 7. KrProgressLog を生成（checkbox 用）
        auto logResult = KrProgressLog::createCheckbox(
            logIdResult.value(),
            krIdResult.value(),
            recorderIdResult.value(),
            input.completed,
            input.note
        );
        if (!logResult) {
            return Result<KeyResultOutput>::err(logResult.error());
        }

        // 8. ドメイン操作：進捗更新（進捗種別一致チェック付き）
        KeyResultProgress newProgress = cp;
        auto updateResult = kr.updateProgress(std::move(newProgress), std::move(logResult.value()));
        if (!updateResult) {
            return Result<KeyResultOutput>::err(updateResult.error());
        }

        // 9. 永続化
        auto saveResult = repo_->save(kr);
        if (!saveResult) {
            return Result<KeyResultOutput>::err(saveResult.error());
        }

        return Result<KeyResultOutput>::ok(KeyResultOutput::from(kr));
    }

    // ──────────────────────────────────────────────────────────────────────
    // DeleteKeyResult — ID で KR を削除する
    //
    // 対象が存在しない場合もエラーにしない（冪等性を保証）。
    // ──────────────────────────────────────────────────────────────────────
    Result<void> deleteKeyResult(const DeleteKeyResultInput& input) {
        // KeyResultId を生成（UUID バリデーション）
        auto idResult = KeyResultId::create(input.keyResultId);
        if (!idResult) {
            return Result<void>::err(idResult.error());
        }

        return repo_->remove(idResult.value());
    }

private:
    std::shared_ptr<IKeyResultRepository> repo_;
};
