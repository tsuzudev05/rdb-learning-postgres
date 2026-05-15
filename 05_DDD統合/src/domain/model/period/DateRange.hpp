#pragma once

#include <string>
#include <chrono>
#include "../../common/Result.hpp"

// =============================================================================
// DateRange — 期間（開始日〜終了日）を表す値オブジェクト
// start_date < end_date を保証。イミュータブル。
// 日付は "YYYY-MM-DD" 文字列で保持（PostgreSQL DATE型と対応）。
// =============================================================================
class DateRange {
public:
    static Result<DateRange> create(const std::string& start, const std::string& end) {
        if (!isValidDate(start)) {
            return Result<DateRange>::err("DateRange: 無効な開始日形式です: " + start);
        }
        if (!isValidDate(end)) {
            return Result<DateRange>::err("DateRange: 無効な終了日形式です: " + end);
        }
        if (start >= end) {
            return Result<DateRange>::err(
                "DateRange: 開始日は終了日より前でなければなりません: " + start + " >= " + end
            );
        }
        return Result<DateRange>::ok(DateRange{start, end});
    }

    const std::string& startDate() const { return start_; }
    const std::string& endDate()   const { return end_; }

    bool operator==(const DateRange& other) const {
        return start_ == other.start_ && end_ == other.end_;
    }
    bool operator!=(const DateRange& other) const { return !(*this == other); }

private:
    DateRange(std::string start, std::string end)
        : start_(std::move(start)), end_(std::move(end)) {}

    // YYYY-MM-DD 形式チェック（簡易）
    static bool isValidDate(const std::string& s) {
        if (s.size() != 10) return false;
        if (s[4] != '-' || s[7] != '-') return false;
        for (int i : {0,1,2,3,5,6,8,9}) {
            if (!std::isdigit(s[i])) return false;
        }
        const int month = std::stoi(s.substr(5, 2));
        const int day   = std::stoi(s.substr(8, 2));
        return month >= 1 && month <= 12 && day >= 1 && day <= 31;
    }

    std::string start_;
    std::string end_;
};
