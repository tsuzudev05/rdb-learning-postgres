#pragma once

#include <string>
#include <regex>
#include <algorithm>
#include "../../common/Result.hpp"

// =============================================================================
// Email — メールアドレスを表す値オブジェクト
// RFC5322 簡易検証。小文字正規化済み。イミュータブル。
// =============================================================================
class Email {
public:
    static Result<Email> create(const std::string& value) {
        const std::string normalized = normalize(value);

        if (normalized.empty()) {
            return Result<Email>::err("Email: メールアドレスが空です");
        }
        if (normalized.size() > 255) {
            return Result<Email>::err("Email: メールアドレスが255文字を超えています");
        }
        if (!isValid(normalized)) {
            return Result<Email>::err("Email: 無効なメールアドレス形式です: " + value);
        }

        return Result<Email>::ok(Email{normalized});
    }

    const std::string& value() const { return value_; }

    bool operator==(const Email& other) const { return value_ == other.value_; }
    bool operator!=(const Email& other) const { return !(*this == other); }

private:
    explicit Email(std::string value) : value_(std::move(value)) {}

    // 小文字に正規化
    static std::string normalize(std::string s) {
        std::transform(s.begin(), s.end(), s.begin(), ::tolower);
        return s;
    }

    static bool isValid(const std::string& s) {
        // RFC5322 簡易チェック（local@domain.tld）
        static const std::regex email_regex(
            R"(^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$)"
        );
        return std::regex_match(s, email_regex);
    }

    std::string value_;
};
