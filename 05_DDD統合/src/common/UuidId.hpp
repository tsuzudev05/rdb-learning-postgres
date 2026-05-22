#pragma once

#include <string>
#include <regex>
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
