#pragma once

#include <string>
#include <optional>
#include <utility>
#include "ObjectiveId.hpp"
#include "../period/PeriodId.hpp"
#include "../user/UserId.hpp"
#include "../../../common/Result.hpp"

// =============================================================================
// Objective — 目標エンティティ（集約ルート、OKR の O）
//
// ドメインルール:
//   - title は空にできない（最大 255 文字）
//   - display_order は 0 以上の整数（UI 上の並び順）
//   - owner_id がそのチームメンバーかは ドメインサービス層で検証する
// =============================================================================
class Objective {
public:
    // ファクトリ：新規目標生成
    static Result<Objective> create(
        ObjectiveId                id,
        PeriodId                   periodId,
        UserId                     ownerId,
        std::string                title,
        std::optional<std::string> description  = std::nullopt,
        int                        displayOrder = 0
    ) {
        auto titleResult = validateTitle(title);
        if (!titleResult) return Result<Objective>::err(titleResult.error());

        if (displayOrder < 0) {
            return Result<Objective>::err(
                "Objective: display_order は 0 以上でなければなりません: "
                + std::to_string(displayOrder)
            );
        }

        return Result<Objective>::ok(Objective{
            std::move(id),
            std::move(periodId),
            std::move(ownerId),
            std::move(title),
            std::move(description),
            displayOrder
        });
    }

    // ─── アクセサ ───────────────────────────────────────────
    const ObjectiveId&                  id()           const { return id_; }
    const PeriodId&                     periodId()     const { return periodId_; }
    const UserId&                       ownerId()      const { return ownerId_; }
    const std::string&                  title()        const { return title_; }
    const std::optional<std::string>&   description()  const { return description_; }
    int                                 displayOrder() const { return displayOrder_; }
    const std::string&                  createdAt()    const { return createdAt_; }
    const std::string&                  updatedAt()    const { return updatedAt_; }

    // ─── ドメイン操作 ────────────────────────────────────────
    // タイトル変更
    Result<void> changeTitle(const std::string& newTitle) {
        auto result = validateTitle(newTitle);
        if (!result) return result;
        title_ = newTitle;
        return Result<void>::ok();
    }

    // 説明変更
    void changeDescription(std::optional<std::string> desc) {
        description_ = std::move(desc);
    }

    // オーナー変更（チームメンバー整合性はドメインサービスで保証）
    void changeOwner(UserId newOwnerId) {
        ownerId_ = std::move(newOwnerId);
    }

    // 表示順変更
    Result<void> reorder(int newOrder) {
        if (newOrder < 0) {
            return Result<void>::err(
                "Objective: display_order は 0 以上でなければなりません: "
                + std::to_string(newOrder)
            );
        }
        displayOrder_ = newOrder;
        return Result<void>::ok();
    }

    // DB再構築用：タイムスタンプをセット
    void setTimestamps(std::string createdAt, std::string updatedAt) {
        createdAt_ = std::move(createdAt);
        updatedAt_ = std::move(updatedAt);
    }

    // ─── 等価比較（ID で同一性を判定）───────────────────────
    bool operator==(const Objective& other) const { return id_ == other.id_; }
    bool operator!=(const Objective& other) const { return !(*this == other); }

private:
    Objective(ObjectiveId id, PeriodId periodId, UserId ownerId,
              std::string title, std::optional<std::string> description, int displayOrder)
        : id_(std::move(id))
        , periodId_(std::move(periodId))
        , ownerId_(std::move(ownerId))
        , title_(std::move(title))
        , description_(std::move(description))
        , displayOrder_(displayOrder)
    {}

    static Result<void> validateTitle(const std::string& title) {
        if (title.empty()) {
            return Result<void>::err("Objective: タイトルが空です");
        }
        if (title.size() > 255) {
            return Result<void>::err(
                "Objective: タイトルが255文字を超えています: "
                + std::to_string(title.size()) + "文字"
            );
        }
        return Result<void>::ok();
    }

    ObjectiveId                 id_;
    PeriodId                    periodId_;
    UserId                      ownerId_;
    std::string                 title_;
    std::optional<std::string>  description_;
    int                         displayOrder_;
    std::string                 createdAt_;
    std::string                 updatedAt_;
};
