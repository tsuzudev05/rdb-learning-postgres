#pragma once

#include <string>
#include "../../common/Result.hpp"

// =============================================================================
// Half — 上期 / 下期を表す値オブジェクト
// H1 / H2 のみ許容。イミュータブル。
// =============================================================================
class Half {
public:
    enum class Value { H1, H2 };

    static Result<Half> create(const std::string& value) {
        if (value == "H1") return Result<Half>::ok(Half{Value::H1});
        if (value == "H2") return Result<Half>::ok(Half{Value::H2});
        return Result<Half>::err("Half: 無効な値です: " + value + "（H1 / H2 のみ有効）");
    }

    static Half h1() { return Half{Value::H1}; }
    static Half h2() { return Half{Value::H2}; }

    Value value() const { return value_; }

    std::string toString() const {
        return value_ == Value::H1 ? "H1" : "H2";
    }

    bool operator==(const Half& other) const { return value_ == other.value_; }
    bool operator!=(const Half& other) const { return !(*this == other); }

private:
    explicit Half(Value value) : value_(value) {}
    Value value_;
};
