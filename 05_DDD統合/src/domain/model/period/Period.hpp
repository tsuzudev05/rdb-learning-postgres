#pragma once

#include <string>
#include <utility>
#include "PeriodId.hpp"
#include "Half.hpp"
#include "DateRange.hpp"
#include "../team/TeamId.hpp"
#include "../../../common/Result.hpp"

// =============================================================================
// Period — 半期 OKRサイクルエンティティ（集約ルート）
//
// ドメインルール:
//   - name は空にできない（例: "2025-H1"、最大 50 文字）
//   - half と DateRange は対応していること（ドメインサービスで検証）
//   - Period は一度作成したら基本的に変更しない（不変に近い集約）
// =============================================================================
class Period {
public:
    // ファクトリ：新規期間生成
    static Result<Period> create(
        PeriodId    id,
        TeamId      teamId,
        std::string name,
        Half        half,
        DateRange   dateRange
    ) {
        auto nameResult = validateName(name);
        if (!nameResult) return Result<Period>::err(nameResult.error());

        return Result<Period>::ok(Period{
            std::move(id),
            std::move(teamId),
            std::move(name),
            std::move(half),
            std::move(dateRange)
        });
    }

    // ─── アクセサ ───────────────────────────────────────────
    const PeriodId&    id()        const { return id_; }
    const TeamId&      teamId()    const { return teamId_; }
    const std::string& name()      const { return name_; }
    const Half&        half()      const { return half_; }
    const DateRange&   dateRange() const { return dateRange_; }
    const std::string& createdAt() const { return createdAt_; }
    const std::string& updatedAt() const { return updatedAt_; }

    // DB再構築用：タイムスタンプをセット
    void setTimestamps(std::string createdAt, std::string updatedAt) {
        createdAt_ = std::move(createdAt);
        updatedAt_ = std::move(updatedAt);
    }

    // ─── 等価比較（ID で同一性を判定）───────────────────────
    bool operator==(const Period& other) const { return id_ == other.id_; }
    bool operator!=(const Period& other) const { return !(*this == other); }

private:
    Period(PeriodId id, TeamId teamId, std::string name, Half half, DateRange dateRange)
        : id_(std::move(id))
        , teamId_(std::move(teamId))
        , name_(std::move(name))
        , half_(std::move(half))
        , dateRange_(std::move(dateRange))
    {}

    static Result<void> validateName(const std::string& name) {
        if (name.empty()) {
            return Result<void>::err("Period: 期間名が空です");
        }
        if (name.size() > 50) {
            return Result<void>::err(
                "Period: 期間名が50文字を超えています: " + std::to_string(name.size()) + "文字"
            );
        }
        return Result<void>::ok();
    }

    PeriodId    id_;
    TeamId      teamId_;
    std::string name_;
    Half        half_;
    DateRange   dateRange_;
    std::string createdAt_;
    std::string updatedAt_;
};
