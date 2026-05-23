#pragma once

#include <string>
#include <regex>
#include <random>
#include <sstream>
#include <iomanip>
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

    // ファクトリ：UUID v4 を新規生成
    // アプリケーション層（UserUseCase）で新規ユーザー作成時に使用する。
    static Result<UserId> generate() {
        std::random_device rd;
        std::mt19937_64 gen(rd());
        std::uniform_int_distribution<uint64_t> dist(0, UINT64_MAX);

        uint64_t hi = dist(gen);
        uint64_t lo = dist(gen);

        // UUID v4: version=4, variant=10xx
        hi = (hi & 0xFFFFFFFFFFFF0FFFULL) | 0x0000000000004000ULL;
        lo = (lo & 0x3FFFFFFFFFFFFFFFULL) | 0x8000000000000000ULL;

        std::ostringstream oss;
        oss << std::hex << std::setfill('0')
            << std::setw(8) << ((hi >> 32) & 0xFFFFFFFF) << '-'
            << std::setw(4) << ((hi >> 16) & 0xFFFF)     << '-'
            << std::setw(4) << (hi & 0xFFFF)              << '-'
            << std::setw(4) << ((lo >> 48) & 0xFFFF)      << '-'
            << std::setw(12) << (lo & 0x0000FFFFFFFFFFFFULL);

        return Result<UserId>::ok(UserId{oss.str()});
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
