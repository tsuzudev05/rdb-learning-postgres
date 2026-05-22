#pragma once

#include <string>
#include "../../../common/Result.hpp"

// =============================================================================
// Role — チームメンバーのロールを表す値オブジェクト
// admin / member のみ許容。イミュータブル。
// =============================================================================
class Role {
public:
    enum class Value { Admin, Member };

    static Result<Role> create(const std::string& value) {
        if (value == "admin")  return Result<Role>::ok(Role{Value::Admin});
        if (value == "member") return Result<Role>::ok(Role{Value::Member});
        return Result<Role>::err("Role: 無効なロールです: " + value + "（admin / member のみ有効）");
    }

    static Role admin()  { return Role{Value::Admin}; }
    static Role member() { return Role{Value::Member}; }

    Value value() const { return value_; }

    bool isAdmin()  const { return value_ == Value::Admin; }
    bool isMember() const { return value_ == Value::Member; }

    // DB保存・表示用
    std::string toString() const {
        return value_ == Value::Admin ? "admin" : "member";
    }

    bool operator==(const Role& other) const { return value_ == other.value_; }
    bool operator!=(const Role& other) const { return !(*this == other); }

private:
    explicit Role(Value value) : value_(value) {}
    Value value_;
};
