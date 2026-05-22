#pragma once

#include <stdexcept>
#include <string>
#include <variant>

// Result<T> -- domain layer error handling type
// Self-implementation of std::expected (C++23). No external dependencies.
//
// Design notes:
//   - static ok() / err() factories use in_place construction to avoid
//     requiring T to be default-constructible (important for value objects
//     whose constructors are private).
//   - Result<void> specialization: C++ does not allow a static and non-static
//     member function with the same name, so instance ok() is omitted;
//     use operator bool() to check success instead.

struct DomainError {
    std::string message;
    explicit DomainError(std::string msg) : message(std::move(msg)) {}
};

// Result<T> -- holds success value or DomainError
template <typename T>
class Result {
public:
    static Result ok(T value) {
        return Result{std::variant<T, DomainError>{
            std::in_place_type<T>, std::move(value)}};
    }

    static Result err(std::string message) {
        return Result{std::variant<T, DomainError>{
            std::in_place_type<DomainError>, std::move(message)}};
    }

    bool ok() const {
        return std::holds_alternative<T>(storage_);
    }
    explicit operator bool() const { return ok(); }

    const T& value() const {
        if (!ok()) throw std::logic_error("Result is error: " + error());
        return std::get<T>(storage_);
    }
    T& value() {
        if (!ok()) throw std::logic_error("Result is error: " + error());
        return std::get<T>(storage_);
    }

    const std::string& error() const {
        return std::get<DomainError>(storage_).message;
    }

private:
    explicit Result(std::variant<T, DomainError> storage)
        : storage_(std::move(storage)) {}

    std::variant<T, DomainError> storage_;
};

// Result<void> -- success or DomainError (no value variant)
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
        r.error_message_ = std::move(message);
        return r;
    }

    explicit operator bool() const { return success_; }

    const std::string& error() const { return error_message_; }

private:
    bool        success_       = false;
    std::string error_message_;
};
