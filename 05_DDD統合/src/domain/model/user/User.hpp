#pragma once

#include <string>
#include <utility>
#include "UserId.hpp"
#include "Email.hpp"
#include "../../../common/Result.hpp"

// =============================================================================
// User — ユーザーエンティティ（集約ルート）
//
// ドメインルール:
//   - name は空にできない（最大 100 文字）
//   - email は Email 値オブジェクトで検証済み
//   - password_hash は外部（認証サービス等）でハッシュ化済みの文字列を受け取る
//   - created_at / updated_at は DB 側で管理するため任意フィールド
// =============================================================================
class User {
public:
    // ファクトリ：新規ユーザー生成
    static Result<User> create(
        UserId          id,
        std::string     name,
        Email           email,
        std::string     passwordHash
    ) {
        auto nameResult = validateName(name);
        if (!nameResult) return Result<User>::err(nameResult.error());

        return Result<User>::ok(User{
            std::move(id),
            std::move(name),
            std::move(email),
            std::move(passwordHash)
        });
    }

    // ─── アクセサ ───────────────────────────────────────────
    const UserId&      id()           const { return id_; }
    const std::string& name()         const { return name_; }
    const Email&       email()        const { return email_; }
    const std::string& passwordHash() const { return passwordHash_; }
    const std::string& createdAt()    const { return createdAt_; }
    const std::string& updatedAt()    const { return updatedAt_; }

    // ─── ドメイン操作 ────────────────────────────────────────
    // 名前変更
    Result<void> changeName(const std::string& newName) {
        auto result = validateName(newName);
        if (!result) return result;
        name_ = newName;
        return Result<void>::ok();
    }

    // パスワードハッシュ更新（認証サービスからハッシュ化済み文字列を受け取る）
    Result<void> changePasswordHash(const std::string& newHash) {
        if (newHash.empty()) {
            return Result<void>::err("User: パスワードハッシュが空です");
        }
        passwordHash_ = newHash;
        return Result<void>::ok();
    }

    // DB再構築用：タイムスタンプをセット（インフラ層からのみ呼び出す）
    void setTimestamps(std::string createdAt, std::string updatedAt) {
        createdAt_ = std::move(createdAt);
        updatedAt_ = std::move(updatedAt);
    }

    // ─── 等価比較（ID で同一性を判定）───────────────────────
    bool operator==(const User& other) const { return id_ == other.id_; }
    bool operator!=(const User& other) const { return !(*this == other); }

private:
    User(UserId id, std::string name, Email email, std::string passwordHash)
        : id_(std::move(id))
        , name_(std::move(name))
        , email_(std::move(email))
        , passwordHash_(std::move(passwordHash))
    {}

    static Result<void> validateName(const std::string& name) {
        if (name.empty()) {
            return Result<void>::err("User: 名前が空です");
        }
        if (name.size() > 100) {
            return Result<void>::err(
                "User: 名前が100文字を超えています: " + std::to_string(name.size()) + "文字"
            );
        }
        return Result<void>::ok();
    }

    UserId      id_;
    std::string name_;
    Email       email_;
    std::string passwordHash_;
    std::string createdAt_;
    std::string updatedAt_;
};
