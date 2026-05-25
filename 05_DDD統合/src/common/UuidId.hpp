#pragma once

#include <string>
#include <regex>
#include <random>
#include <sstream>
#include <iomanip>
#include "Result.hpp"

// =============================================================================
// UuidId<Tag> — UUID v4 に基づく型安全な ID 値オブジェクト テンプレート
//
// 各エンティティの ID はタグ型でパラメータ化することで、
// 同じ UUID 形式でも TeamId と UserId を混在させてしまうコンパイルエラーを防ぐ。
//
// 使い方:
//   struct TeamTag {};
//   using TeamId = UuidId<TeamTag>;
//
//   auto result = TeamId::create("550e8400-e29b-41d4-a716-446655440000");
//   if (!result) { /* エラー */ }
//   TeamId id = result.value();
// =============================================================================
template <typename Tag>
class UuidId {
public:
    // ファクトリ：UUID v4 形式を検証して生成
    static Result<UuidId<Tag>> create(const std::string& value) {
        if (!isValidUuid(value)) {
            return Result<UuidId<Tag>>::err(
                "UuidId: 無効なUUID形式です: " + value
            );
        }
        return Result<UuidId<Tag>>::ok(UuidId<Tag>{value});
    }

    // ファクトリ：UUID v4 を新規生成
    // アプリケーション層（UseCase）で新規エンティティ作成時に使用する。
    static Result<UuidId<Tag>> generate() {
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

        return Result<UuidId<Tag>>::ok(UuidId<Tag>{oss.str()});
    }

    // 値の取得（DB 保存・比較用）
    const std::string& value() const { return value_; }

    // 等価比較（同じタグ型のみ比較可能）
    bool operator==(const UuidId<Tag>& other) const { return value_ == other.value_; }
    bool operator!=(const UuidId<Tag>& other) const { return !(*this == other); }
    bool operator< (const UuidId<Tag>& other) const { return value_ < other.value_; }

private:
    explicit UuidId(std::string value) : value_(std::move(value)) {}

    // UUID v4: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
    static bool isValidUuid(const std::string& s) {
        static const std::regex uuid_regex(
            "^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$",
            std::regex::icase
        );
        return std::regex_match(s, uuid_regex);
    }

    std::string value_;
};
