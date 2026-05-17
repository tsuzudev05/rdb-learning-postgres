#pragma once

#include <string>
#include <vector>
#include <utility>
#include "KeyResultId.hpp"
#include "KeyResultProgress.hpp"
#include "KrProgressLog.hpp"
#include "../objective/ObjectiveId.hpp"
#include "../user/UserId.hpp"
#include "../../../common/Result.hpp"

// =============================================================================
// KeyResult — 主要な結果エンティティ（集約ルート、OKR の KR）
//
// ドメインルール:
//   - title は空にできない（最大 255 文字）
//   - progress は KeyResultProgress（NumericProgress / CheckboxProgress）
//   - display_order は 0 以上の整数
//   - 進捗更新時は KrProgressLog を追記し、現在の progress も更新する
//   - owner_id がチームメンバーかはドメインサービス層で検証する
// =============================================================================
class KeyResult {
public:
    // ファクトリ：新規 KeyResult 生成
    static Result<KeyResult> create(
        KeyResultId         id,
        ObjectiveId         objectiveId,
        UserId              ownerId,
        std::string         title,
        KeyResultProgress   progress,
        int                 displayOrder = 0
    ) {
        auto titleResult = validateTitle(title);
        if (!titleResult) return Result<KeyResult>::err(titleResult.error());

        if (displayOrder < 0) {
            return Result<KeyResult>::err(
                "KeyResult: display_order は 0 以上でなければなりません: "
                + std::to_string(displayOrder)
            );
        }

        return Result<KeyResult>::ok(KeyResult{
            std::move(id),
            std::move(objectiveId),
            std::move(ownerId),
            std::move(title),
            std::move(progress),
            displayOrder
        });
    }

    // ─── アクセサ ───────────────────────────────────────────
    const KeyResultId&              id()              const { return id_; }
    const ObjectiveId&              objectiveId()     const { return objectiveId_; }
    const UserId&                   ownerId()         const { return ownerId_; }
    const std::string&              title()           const { return title_; }
    const KeyResultProgress&        progress()        const { return progress_; }
    int                             displayOrder()    const { return displayOrder_; }
    const std::vector<KrProgressLog>& progressLogs()  const { return progressLogs_; }
    const std::string&              createdAt()       const { return createdAt_; }
    const std::string&              updatedAt()       const { return updatedAt_; }

    // 達成率（0.0〜1.0）
    double achievementRate() const {
        return getAchievementRate(progress_);
    }

    // KR 種別文字列（"numeric" / "checkbox"）
    std::string krType() const {
        return getKrType(progress_);
    }

    // ─── ドメイン操作 ────────────────────────────────────────
    // タイトル変更
    Result<void> changeTitle(const std::string& newTitle) {
        auto result = validateTitle(newTitle);
        if (!result) return result;
        title_ = newTitle;
        return Result<void>::ok();
    }

    // オーナー変更（チームメンバー整合性はドメインサービスで保証）
    void changeOwner(UserId newOwnerId) {
        ownerId_ = std::move(newOwnerId);
    }

    // 表示順変更
    Result<void> reorder(int newOrder) {
        if (newOrder < 0) {
            return Result<void>::err(
                "KeyResult: display_order は 0 以上でなければなりません: "
                + std::to_string(newOrder)
            );
        }
        displayOrder_ = newOrder;
        return Result<void>::ok();
    }

    // 進捗更新（progress を更新し、ログを追記）
    Result<void> updateProgress(KeyResultProgress newProgress, KrProgressLog log) {
        // 進捗種別が変わっていないことを確認
        if (getKrType(newProgress) != getKrType(progress_)) {
            return Result<void>::err(
                "KeyResult: 進捗種別を変更することはできません（"
                + getKrType(progress_) + " → " + getKrType(newProgress) + "）"
            );
        }
        progress_ = std::move(newProgress);
        progressLogs_.push_back(std::move(log));
        return Result<void>::ok();
    }

    // DB再構築用：進捗ログを一括セット（インフラ層からのみ呼び出す）
    void setProgressLogs(std::vector<KrProgressLog> logs) {
        progressLogs_ = std::move(logs);
    }

    // DB再構築用：タイムスタンプをセット
    void setTimestamps(std::string createdAt, std::string updatedAt) {
        createdAt_ = std::move(createdAt);
        updatedAt_ = std::move(updatedAt);
    }

    // ─── 等価比較（ID で同一性を判定）───────────────────────
    bool operator==(const KeyResult& other) const { return id_ == other.id_; }
    bool operator!=(const KeyResult& other) const { return !(*this == other); }

private:
    KeyResult(KeyResultId id, ObjectiveId objectiveId, UserId ownerId,
              std::string title, KeyResultProgress progress, int displayOrder)
        : id_(std::move(id))
        , objectiveId_(std::move(objectiveId))
        , ownerId_(std::move(ownerId))
        , title_(std::move(title))
        , progress_(std::move(progress))
        , displayOrder_(displayOrder)
    {}

    static Result<void> validateTitle(const std::string& title) {
        if (title.empty()) {
            return Result<void>::err("KeyResult: タイトルが空です");
        }
        if (title.size() > 255) {
            return Result<void>::err(
                "KeyResult: タイトルが255文字を超えています: "
                + std::to_string(title.size()) + "文字"
            );
        }
        return Result<void>::ok();
    }

    KeyResultId              id_;
    ObjectiveId              objectiveId_;
    UserId                   ownerId_;
    std::string              title_;
    KeyResultProgress        progress_;
    int                      displayOrder_;
    std::vector<KrProgressLog> progressLogs_;
    std::string              createdAt_;
    std::string              updatedAt_;
};
