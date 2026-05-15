#pragma once

#include <stdexcept>
#include <string>
#include <variant>

// =============================================================================
// Result<T> — ドメイン層共通エラーハンドリング型
// std::expected (C++23) の自前実装。外部依存ゼロ。
//
// 使い方:
//   Result<User> r = UserFactory::create("alice@example.com");
//   if (!r) { std::cerr << r.error() << "\n"; }
//   User user = r.value();
//
//   Result<void> r2 = repo.save(user);
//   if (!r2) { ... }
// =============================================================================

// -----------------------------------------------------------------------------
// DomainError — ドメイン層のエラー型
// -----------------------------------------------------------------------------
struct DomainError {
    std::string message;
    explicit DomainError(std::string msg) : message(std::move(msg)) {}
};

// -----------------------------------------------------------------------------
// Result<T> — 成功値 or DomainError を保持する型
// -----------------------------------------------------------------------------
template <typename T>
class Result {
public:
    // 成功
    static Result ok(T value) {
        Result r;
        r.storage_ = std::move(value);
        return r;
    }

    // 失敗
    static Result err(std::string message) {
        Result r;
        r.storage_ = DomainError{std::move(message)};
        return r;
    }

    // 成功判定
    bool ok() const {
        return std::holds_alternative<T>(storage_);
    }
    explicit operator bool() const { return ok(); }

    // 成功値の取得（失敗時は例外）
    const T& value() const {
        if (!ok()) throw std::logic_error("Result is error: " + error());
        return std::get<T>(storage_);
    }
    T& value() {
        if (!ok()) throw std::logic_error("Result is error: " + error());
        return std::get<T>(storage_);
    }

    // エラーメッセージの取得
    const std::string& error() const {
        return std::get<DomainError>(storage_).message;
    }

private:
    std::variant<T, DomainError> storage_;
};

// -----------------------------------------------------------------------------
// Result<void> — 成功 or DomainError（値なし版）
// -----------------------------------------------------------------------------
template <>
class Result<void> {
public:
    static Result ok() {
        Result r;
        r.success_ = true;
        return r;
    }

    static Result err(std::string message) {
        Result r;
        r.success_ = false;
        r.error_ = std::move(message);
        return r;
    }

    bool ok() const { return success_; }
    explicit operator bool() const { return ok(); }

    const std::string& error() const { return error_; }

private:
    bool success_ = false;
    std::string error_;
};
