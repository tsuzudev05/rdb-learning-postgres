#pragma once

#include <memory>
#include <string>
#include <optional>
#include <vector>
#include "../../domain/model/objective/Objective.hpp"
#include "../../domain/model/objective/ObjectiveId.hpp"
#include "../../domain/model/period/PeriodId.hpp"
#include "../../domain/model/user/UserId.hpp"
#include "../../domain/repository/IObjectiveRepository.hpp"
#include "../../common/Result.hpp"

// =============================================================================
// ObjectiveUseCase — 目標ユースケース（アプリケーション層）
//
// ドメイン層の IObjectiveRepository に依存注入（DI）パターンで依存する。
// インフラ層（libpqxx）への直接依存を持たない。
//
// 提供するユースケース:
//   - CreateObjective       : 新規目標を作成する
//   - GetObjective          : ID で目標を1件取得する
//   - ListObjectivesByPeriod: 期間IDに紐づく目標一覧を取得する（display_order 昇順）
//   - ListObjectivesByOwner : オーナーIDに紐づく目標一覧を取得する
//   - DeleteObjective       : ID で目標を削除する
//
// 注意:
//   owner_id がそのチームのメンバーかどうかはドメインサービス層で検証するが、
//   現フェーズでは UseCase 内でバリデーションをスキップし、DBに委ねる。
//
// エラーハンドリング:
//   すべての操作は Result<T> を返す。例外は使わない。
// =============================================================================

// ─── 入力 DTO ──────────────────────────────────────────────────────────────

struct CreateObjectiveInput {
    std::string periodId;                     // UUID 文字列
    std::string ownerId;                      // UUID 文字列
    std::string title;
    std::optional<std::string> description;   // 省略可
    int displayOrder = 0;                     // UI 上の並び順（0以上）
};

struct GetObjectiveInput {
    std::string objectiveId;  // UUID 文字列
};

struct DeleteObjectiveInput {
    std::string objectiveId;  // UUID 文字列
};

struct ListObjectivesByPeriodInput {
    std::string periodId;     // UUID 文字列
};

struct ListObjectivesByOwnerInput {
    std::string ownerId;      // UUID 文字列
};

// ─── 出力 DTO ──────────────────────────────────────────────────────────────

struct ObjectiveOutput {
    std::string id;
    std::string periodId;
    std::string ownerId;
    std::string title;
    std::optional<std::string> description;
    int displayOrder;
    std::string createdAt;
    std::string updatedAt;

    // Objective エンティティから変換するファクトリ
    static ObjectiveOutput from(const Objective& obj) {
        return ObjectiveOutput{
            obj.id().value(),
            obj.periodId().value(),
            obj.ownerId().value(),
            obj.title(),
            obj.description(),
            obj.displayOrder(),
            obj.createdAt(),
            obj.updatedAt()
        };
    }
};

// ─── ObjectiveUseCase ───────────────────────────────────────────────────────

class ObjectiveUseCase {
public:
    // コンストラクタ：IObjectiveRepository を DI で受け取る（所有権は共有しない）
    explicit ObjectiveUseCase(std::shared_ptr<IObjectiveRepository> repo)
        : repo_(std::move(repo)) {}

    // ──────────────────────────────────────────────────────────────────────
    // CreateObjective — 新規目標を作成する
    //
    // 処理フロー:
    //   1. 入力値を各値オブジェクトに変換（バリデーション含む）
    //   2. ObjectiveId を新規生成
    //   3. Objective エンティティを生成
    //   4. Repository#save で永続化
    // ──────────────────────────────────────────────────────────────────────
    Result<ObjectiveOutput> createObjective(const CreateObjectiveInput& input) {
        // 1. PeriodId を生成（UUID バリデーション）
        auto periodIdResult = PeriodId::create(input.periodId);
        if (!periodIdResult) {
            return Result<ObjectiveOutput>::err(periodIdResult.error());
        }

        // 2. UserId（ownerId）を生成（UUID バリデーション）
        auto ownerIdResult = UserId::create(input.ownerId);
        if (!ownerIdResult) {
            return Result<ObjectiveOutput>::err(ownerIdResult.error());
        }

        // 3. ObjectiveId を新規生成
        auto idResult = ObjectiveId::generate();
        if (!idResult) {
            return Result<ObjectiveOutput>::err(idResult.error());
        }

        // 4. Objective エンティティを生成（title・displayOrder バリデーション含む）
        auto objResult = Objective::create(
            idResult.value(),
            periodIdResult.value(),
            ownerIdResult.value(),
            input.title,
            input.description,
            input.displayOrder
        );
        if (!objResult) {
            return Result<ObjectiveOutput>::err(objResult.error());
        }

        // 5. 永続化
        auto saveResult = repo_->save(objResult.value());
        if (!saveResult) {
            return Result<ObjectiveOutput>::err(saveResult.error());
        }

        return Result<ObjectiveOutput>::ok(ObjectiveOutput::from(objResult.value()));
    }

    // ──────────────────────────────────────────────────────────────────────
    // GetObjective — ID で目標を1件取得する
    //
    // 対象が存在しない場合はエラー（NotFound）を返す。
    // ──────────────────────────────────────────────────────────────────────
    Result<ObjectiveOutput> getObjective(const GetObjectiveInput& input) {
        // ObjectiveId を生成（UUID バリデーション）
        auto idResult = ObjectiveId::create(input.objectiveId);
        if (!idResult) {
            return Result<ObjectiveOutput>::err(idResult.error());
        }

        // Repository から取得
        auto findResult = repo_->findById(idResult.value());
        if (!findResult) {
            return Result<ObjectiveOutput>::err(findResult.error());
        }
        if (!findResult.value().has_value()) {
            return Result<ObjectiveOutput>::err(
                "GetObjective: 目標が見つかりません: " + input.objectiveId
            );
        }

        return Result<ObjectiveOutput>::ok(ObjectiveOutput::from(findResult.value().value()));
    }

    // ──────────────────────────────────────────────────────────────────────
    // ListObjectivesByPeriod — 期間IDに紐づく目標一覧を取得する（display_order 昇順）
    // ──────────────────────────────────────────────────────────────────────
    Result<std::vector<ObjectiveOutput>> listObjectivesByPeriod(
        const ListObjectivesByPeriodInput& input
    ) {
        // PeriodId を生成（UUID バリデーション）
        auto idResult = PeriodId::create(input.periodId);
        if (!idResult) {
            return Result<std::vector<ObjectiveOutput>>::err(idResult.error());
        }

        auto findResult = repo_->findByPeriodId(idResult.value());
        if (!findResult) {
            return Result<std::vector<ObjectiveOutput>>::err(findResult.error());
        }

        std::vector<ObjectiveOutput> outputs;
        outputs.reserve(findResult.value().size());
        for (const auto& obj : findResult.value()) {
            outputs.push_back(ObjectiveOutput::from(obj));
        }

        return Result<std::vector<ObjectiveOutput>>::ok(std::move(outputs));
    }

    // ──────────────────────────────────────────────────────────────────────
    // ListObjectivesByOwner — オーナーIDに紐づく目標一覧を取得する
    // ──────────────────────────────────────────────────────────────────────
    Result<std::vector<ObjectiveOutput>> listObjectivesByOwner(
        const ListObjectivesByOwnerInput& input
    ) {
        // UserId（ownerId）を生成（UUID バリデーション）
        auto idResult = UserId::create(input.ownerId);
        if (!idResult) {
            return Result<std::vector<ObjectiveOutput>>::err(idResult.error());
        }

        auto findResult = repo_->findByOwnerId(idResult.value());
        if (!findResult) {
            return Result<std::vector<ObjectiveOutput>>::err(findResult.error());
        }

        std::vector<ObjectiveOutput> outputs;
        outputs.reserve(findResult.value().size());
        for (const auto& obj : findResult.value()) {
            outputs.push_back(ObjectiveOutput::from(obj));
        }

        return Result<std::vector<ObjectiveOutput>>::ok(std::move(outputs));
    }

    // ──────────────────────────────────────────────────────────────────────
    // DeleteObjective — ID で目標を削除する
    //
    // 対象が存在しない場合もエラーにしない（冪等性を保証）。
    // ──────────────────────────────────────────────────────────────────────
    Result<void> deleteObjective(const DeleteObjectiveInput& input) {
        // ObjectiveId を生成（UUID バリデーション）
        auto idResult = ObjectiveId::create(input.objectiveId);
        if (!idResult) {
            return Result<void>::err(idResult.error());
        }

        return repo_->remove(idResult.value());
    }

private:
    std::shared_ptr<IObjectiveRepository> repo_;
};
