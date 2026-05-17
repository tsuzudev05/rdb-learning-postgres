#pragma once

#include <string>
#include <regex>
#include "../../../common/Result.hpp"

// =============================================================================
// UserId — ユーザーIDを表す値オブジェクト
// UUID v4 形式のみ受け付ける。イミュータブル。
// =============================================================================
class UserId {
public:
    // ファクトリ：バリデーション付き生成
    static Result<UserId> create(const std::string& value) {
        if (!isValidUuid(value)) {
            return Result<UserId>::err("UserId: 無効なUUID形式です: " + value);
        }
        return Result<UserId>::ok(UserId{value});
    }

    // 値の取得
    const std::string& value() const { return value_; }

    // 等価比較
    bool operator==(const UserId& other) const { return value_ == other.value_; }
    bool operator!=(const UserId& other) const { return !(*this == other); }
    bool operator<(const UserId& other)  const { return value_ < other.value_; }

private:
    explicit UserId(std::string value) : value_(std::move(value)) {}

    static bool isValidUuid(const std::string& s) {
        // UUID v4: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
        static const std::regex uuid_regex(
            "^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$",
            std::regex::icase
        );
        return std::regex_match(s, uuid_regex);
    }

    std::string value_;
};
