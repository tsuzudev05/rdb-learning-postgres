#pragma once

#include <string>
#include <optional>
#include <utility>
#include "KrProgressLogId.hpp"
#include "KeyResultId.hpp"
#include "../user/UserId.hpp"
#include "../../../common/Result.hpp"

// =============================================================================
// KrProgressLog — 進捗ログエンティティ（KeyResult 集約内）
//
// ドメインルール:
//   - numeric KR の場合: value に数値を設定し、completed は nullopt
//   - checkbox KR の場合: completed に bool を設定し、value は nullopt
//   - value と completed のどちらか一方のみが有効でなければならない
//   - recorded_at は記録時点のタイムスタンプ（DB 側で DEFAULT NOW()）
// =============================================================================
class KrProgressLog {
public:
    // ファクトリ：数値進捗ログ生成（numeric KR 用）
    static Result<KrProgressLog> createNumeric(
        KrProgressLogId            id,
        KeyResultId                keyResultId,
        UserId                     recordedBy,
        double                     value,
        std::optional<std::string> note        = std::nullopt,
        std::string                recordedAt  = ""
    ) {
        return Result<KrProgressLog>::ok(KrProgressLog{
            std::move(id),
            std::move(keyResultId),
            std::move(recordedBy),
            std::optional<double>{value},
            std::nullopt,
            std::move(note),
            std::move(recordedAt)
        });
    }

    // ファクトリ：完了フラグ進捗ログ生成（checkbox KR 用）
    static Result<KrProgressLog> createCheckbox(
        KrProgressLogId            id,
        KeyResultId                keyResultId,
        UserId                     recordedBy,
        bool                       completed,
        std::optional<std::string> note       = std::nullopt,
        std::string                recordedAt = ""
    ) {
        return Result<KrProgressLog>::ok(KrProgressLog{
            std::move(id),
            std::move(keyResultId),
            std::move(recordedBy),
            std::nullopt,
            std::optional<bool>{completed},
            std::move(note),
            std::move(recordedAt)
        });
    }

    // ─── アクセサ ───────────────────────────────────────────
    const KrProgressLogId&          id()           const { return id_; }
    const KeyResultId&              keyResultId()  const { return keyResultId_; }
    const UserId&                   recordedBy()   const { return recordedBy_; }
    const std::optional<double>&    value()        const { return value_; }
    const std::optional<bool>&      completed()    const { return completed_; }
    const std::optional<std::string>& note()       const { return note_; }
    const std::string&              recordedAt()   const { return recordedAt_; }

    // 進捗種別の判定
    bool isNumeric()   const { return value_.has_value(); }
    bool isCheckbox()  const { return completed_.has_value(); }

    // ─── 等価比較（ID で同一性を判定）───────────────────────
    bool operator==(const KrProgressLog& other) const { return id_ == other.id_; }
    bool operator!=(const KrProgressLog& other) const { return !(*this == other); }

private:
    KrProgressLog(
        KrProgressLogId            id,
        KeyResultId                keyResultId,
        UserId                     recordedBy,
        std::optional<double>      value,
        std::optional<bool>        completed,
        std::optional<std::string> note,
        std::string                recordedAt
    )
        : id_(std::move(id))
        , keyResultId_(std::move(keyResultId))
        , recordedBy_(std::move(recordedBy))
        , value_(std::move(value))
        , completed_(std::move(completed))
        , note_(std::move(note))
        , recordedAt_(std::move(recordedAt))
    {}

    KrProgressLogId            id_;
    KeyResultId                keyResultId_;
    UserId                     recordedBy_;
    std::optional<double>      value_;      // numeric KR の記録値
    std::optional<bool>        completed_;  // checkbox KR の記録値
    std::optional<std::string> note_;
    std::string                recordedAt_;
};
