#pragma once

#include <memory>
#include <string>
#include <optional>
#include <vector>
#include "../../domain/model/user/User.hpp"
#include "../../domain/model/user/UserId.hpp"
#include "../../domain/model/user/Email.hpp"
#include "../../domain/repository/IUserRepository.hpp"
#include "../../common/Result.hpp"

// =============================================================================
// UserUseCase — ユーザーユースケース（アプリケーション層）
//
// ドメイン層の IUserRepository に依存注入（DI）パターンで依存する。
// インフラ層（libpqxx / pgx）への直接依存を持たない。
//
// 提供するユースケース:
//   - CreateUser  : 新規ユーザーを作成する
//   - GetUser     : ID でユーザーを1件取得する
//   - ListUsers   : 全ユーザー一覧を取得する
//   - DeleteUser  : ID でユーザーを削除する
//
// エラーハンドリング:
//   すべての操作は Result<T> を返す。例外は使わない。
// =============================================================================

// ─── 入力 DTO ──────────────────────────────────────────────────────────────

struct CreateUserInput {
    std::string name;
    std::string email;
    std::string passwordHash;  // 認証サービス等でハッシュ化済みの文字列を渡す
};

struct GetUserInput {
    std::string userId;  // UUID 文字列
};

struct DeleteUserInput {
    std::string userId;  // UUID 文字列
};

// ─── 出力 DTO ──────────────────────────────────────────────────────────────

struct UserOutput {
    std::string id;
    std::string name;
    std::string email;
    std::string createdAt;
    std::string updatedAt;

    // User エンティティから変換するファクトリ
    static UserOutput from(const User& user) {
        return UserOutput{
            user.id().value(),
            user.name(),
            user.email().value(),
            user.createdAt(),
            user.updatedAt()
        };
    }
};

// ─── UserUseCase ───────────────────────────────────────────────────────────

class UserUseCase {
public:
    // コンストラクタ：IUserRepository を DI で受け取る（所有権は共有しない）
    explicit UserUseCase(std::shared_ptr<IUserRepository> repo)
        : repo_(std::move(repo)) {}

    // ──────────────────────────────────────────────────────────────────────
    // CreateUser — 新規ユーザーを作成する
    //
    // 処理フロー:
    //   1. Email の重複チェック（同一メールアドレスが存在する場合はエラー）
    //   2. UserId を新規生成
    //   3. User エンティティを生成
    //   4. Repository#save で永続化
    // ──────────────────────────────────────────────────────────────────────
    Result<UserOutput> createUser(const CreateUserInput& input) {
        // 1. Email 値オブジェクトを生成（バリデーション含む）
        auto emailResult = Email::create(input.email);
        if (!emailResult) {
            return Result<UserOutput>::err(emailResult.error());
        }
        const Email& email = emailResult.value();

        // 2. メールアドレス重複チェック
        auto existsResult = repo_->findByEmail(email);
        if (!existsResult) {
            return Result<UserOutput>::err(existsResult.error());
        }
        if (existsResult.value().has_value()) {
            return Result<UserOutput>::err(
                "CreateUser: メールアドレスは既に登録されています: " + input.email
            );
        }

        // 3. UserId を新規生成
        auto idResult = UserId::generate();
        if (!idResult) {
            return Result<UserOutput>::err(idResult.error());
        }

        // 4. User エンティティを生成
        auto userResult = User::create(
            idResult.value(),
            input.name,
            email,
            input.passwordHash
        );
        if (!userResult) {
            return Result<UserOutput>::err(userResult.error());
        }

        // 5. 永続化
        auto saveResult = repo_->save(userResult.value());
        if (!saveResult) {
            return Result<UserOutput>::err(saveResult.error());
        }

        return Result<UserOutput>::ok(UserOutput::from(userResult.value()));
    }

    // ──────────────────────────────────────────────────────────────────────
    // GetUser — ID でユーザーを1件取得する
    //
    // 対象が存在しない場合はエラー（NotFound）を返す。
    // ──────────────────────────────────────────────────────────────────────
    Result<UserOutput> getUser(const GetUserInput& input) {
        // UserId を生成（UUID バリデーション）
        auto idResult = UserId::create(input.userId);
        if (!idResult) {
            return Result<UserOutput>::err(idResult.error());
        }

        // Repository から取得
        auto findResult = repo_->findById(idResult.value());
        if (!findResult) {
            return Result<UserOutput>::err(findResult.error());
        }
        if (!findResult.value().has_value()) {
            return Result<UserOutput>::err(
                "GetUser: ユーザーが見つかりません: " + input.userId
            );
        }

        return Result<UserOutput>::ok(UserOutput::from(findResult.value().value()));
    }

    // ──────────────────────────────────────────────────────────────────────
    // ListUsers — 全ユーザー一覧を取得する
    // ──────────────────────────────────────────────────────────────────────
    Result<std::vector<UserOutput>> listUsers() {
        auto findResult = repo_->findAll();
        if (!findResult) {
            return Result<std::vector<UserOutput>>::err(findResult.error());
        }

        std::vector<UserOutput> outputs;
        outputs.reserve(findResult.value().size());
        for (const auto& user : findResult.value()) {
            outputs.push_back(UserOutput::from(user));
        }

        return Result<std::vector<UserOutput>>::ok(std::move(outputs));
    }

    // ──────────────────────────────────────────────────────────────────────
    // DeleteUser — ID でユーザーを削除する
    //
    // 対象が存在しない場合もエラーにしない（冪等性を保証）。
    // ──────────────────────────────────────────────────────────────────────
    Result<void> deleteUser(const DeleteUserInput& input) {
        // UserId を生成（UUID バリデーション）
        auto idResult = UserId::create(input.userId);
        if (!idResult) {
            return Result<void>::err(idResult.error());
        }

        return repo_->remove(idResult.value());
    }

private:
    std::shared_ptr<IUserRepository> repo_;
};
