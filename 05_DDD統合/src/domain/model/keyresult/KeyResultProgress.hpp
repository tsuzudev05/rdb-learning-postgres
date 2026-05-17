#pragma once

#include <variant>
#include <optional>
#include <string>
#include "../../../common/Result.hpp"

// =============================================================================
// NumericProgress — 数値進捗を表す値オブジェクト
// target > 0、current >= 0 を保証。
// =============================================================================
class NumericProgress {
public:
    static Result<NumericProgress> create(double target, double current) {
        if (target <= 0.0) {
            return Result<NumericProgress>::err(
                "NumericProgress: 目標値は0より大きい必要があります: " + std::to_string(target)
            );
        }
        if (current < 0.0) {
            return Result<NumericProgress>::err(
                "NumericProgress: 現在値は0以上である必要があります: " + std::to_string(current)
            );
        }
        return Result<NumericProgress>::ok(NumericProgress{target, current});
    }

    double targetValue()  const { return target_; }
    double currentValue() const { return current_; }

    // 達成率（0.0〜1.0）
    double achievementRate() const {
        return current_ / target_;
    }

    bool operator==(const NumericProgress& o) const {
        return target_ == o.target_ && current_ == o.current_;
    }

private:
    NumericProgress(double target, double current)
        : target_(target), current_(current) {}
    double target_;
    double current_;
};

// =============================================================================
// CheckboxProgress — 完了フラグを表す値オブジェクト
// =============================================================================
class CheckboxProgress {
public:
    static CheckboxProgress create(bool completed) {
        return CheckboxProgress{completed};
    }

    bool isCompleted() const { return completed_; }

    // 達成率（0.0 or 1.0）
    double achievementRate() const { return completed_ ? 1.0 : 0.0; }

    bool operator==(const CheckboxProgress& o) const {
        return completed_ == o.completed_;
    }

private:
    explicit CheckboxProgress(bool completed) : completed_(completed) {}
    bool completed_;
};

// =============================================================================
// KeyResultProgress — NumericProgress / CheckboxProgress の Union型
// std::variant で型安全に表現。
// =============================================================================
using KeyResultProgress = std::variant<NumericProgress, CheckboxProgress>;

// ユーティリティ：達成率を取得（visitor パターン）
inline double getAchievementRate(const KeyResultProgress& progress) {
    return std::visit([](const auto& p) { return p.achievementRate(); }, progress);
}

// ユーティリティ：種別文字列を取得
inline std::string getKrType(const KeyResultProgress& progress) {
    return std::holds_alternative<NumericProgress>(progress) ? "numeric" : "checkbox";
}
