#pragma once

#include <optional>
#include <vector>
#include "../model/user/User.hpp"
#include "../model/user/UserId.hpp"
#include "../model/user/Email.hpp"
#include "../../common/Result.hpp"

// =============================================================================
// IUserRepository — ユーザーRepositoryインターフェース（純粋仮想）
//
// ドメイン層に属する。インフラ層への依存を持たない。
// 実装は infrastructure/repository/PgUserRepository で行う。
//
// 操作一覧:
//   - findById    : ID でユーザーを1件取得
//   - findByEmail : メールアドレスでユーザーを1件取得
//   - findAll     : 全ユーザーを取得
//   - save        : 新規作成 or 更新（upsert）
//   - remove      : ID でユーザーを削除
// =============================================================================
class IUserRepository {
public:
    virtual ~IUserRepository() = default;

    // ID でユーザーを取得。存在しない場合は std::nullopt を返す。
    virtual Result<std::optional<User>> findById(const UserId& id) const = 0;

    // メールアドレスでユーザーを取得。存在しない場合は std::nullopt を返す。
    virtual Result<std::optional<User>> findByEmail(const Email& email) const = 0;

    // 全ユーザーを取得。
    virtual Result<std::vector<User>> findAll() const = 0;

    // ユーザーを保存（新規作成 or 更新）。
    virtual Result<void> save(const User& user) = 0;

    // ID でユーザーを削除。対象が存在しない場合もエラーにしない。
    virtual Result<void> remove(const UserId& id) = 0;
};
