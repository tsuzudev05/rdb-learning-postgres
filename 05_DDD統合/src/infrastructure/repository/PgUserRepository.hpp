#pragma once

#include <memory>
#include <optional>
#include <vector>
#include <stdexcept>

#include <pqxx/pqxx>

#include "../../domain/repository/IUserRepository.hpp"
#include "../../domain/model/user/User.hpp"
#include "../../domain/model/user/UserId.hpp"
#include "../../domain/model/user/Email.hpp"
#include "../../common/Result.hpp"

// =============================================================================
// PgUserRepository — IUserRepository の PostgreSQL（libpqxx）実装
//
// インフラ層。pqxx::connection を shared_ptr で受け取り、
// 各メソッドでトランザクションを開いて操作する。
//
// SQL 方針:
//   - save(): INSERT ... ON CONFLICT (id) DO UPDATE SET ... (upsert)
//   - remove(): 対象が存在しなくてもエラーにしない
//   - タイムスタンプは DB 側の DEFAULT / トリガーに任せる
// =============================================================================
class PgUserRepository : public IUserRepository {
public:
    explicit PgUserRepository(std::shared_ptr<pqxx::connection> conn)
        : conn_(std::move(conn)) {}

    // -------------------------------------------------------------------------
    // findById
    // -------------------------------------------------------------------------
    Result<std::optional<User>> findById(const UserId& id) const override {
        try {
            pqxx::work tx{*conn_};
            pqxx::result rows = tx.exec_params(
                "SELECT id, name, email, password_hash, "
                "       created_at::TEXT, updated_at::TEXT "
                "FROM users WHERE id = $1",
                id.value()
            );
            tx.commit();

            if (rows.empty()) {
                return Result<std::optional<User>>::ok(std::nullopt);
            }
            auto user = reconstruct(rows[0]);
            if (!user) return Result<std::optional<User>>::err(user.error());
            return Result<std::optional<User>>::ok(
                std::optional<User>{std::move(user.value())}
            );
        } catch (const std::exception& e) {
            return Result<std::optional<User>>::err(
                "PgUserRepository::findById: " + std::string(e.what())
            );
        }
    }

    // -------------------------------------------------------------------------
    // findByEmail
    // -------------------------------------------------------------------------
    Result<std::optional<User>> findByEmail(const Email& email) const override {
        try {
            pqxx::work tx{*conn_};
            pqxx::result rows = tx.exec_params(
                "SELECT id, name, email, password_hash, "
                "       created_at::TEXT, updated_at::TEXT "
                "FROM users WHERE email = $1",
                email.value()
            );
            tx.commit();

            if (rows.empty()) {
                return Result<std::optional<User>>::ok(std::nullopt);
            }
            auto user = reconstruct(rows[0]);
            if (!user) return Result<std::optional<User>>::err(user.error());
            return Result<std::optional<User>>::ok(
                std::optional<User>{std::move(user.value())}
            );
        } catch (const std::exception& e) {
            return Result<std::optional<User>>::err(
                "PgUserRepository::findByEmail: " + std::string(e.what())
            );
        }
    }

    // -------------------------------------------------------------------------
    // findAll
    // -------------------------------------------------------------------------
    Result<std::vector<User>> findAll() const override {
        try {
            pqxx::work tx{*conn_};
            pqxx::result rows = tx.exec(
                "SELECT id, name, email, password_hash, "
                "       created_at::TEXT, updated_at::TEXT "
                "FROM users ORDER BY created_at ASC"
            );
            tx.commit();

            std::vector<User> users;
            users.reserve(rows.size());
            for (const auto& row : rows) {
                auto user = reconstruct(row);
                if (!user) return Result<std::vector<User>>::err(user.error());
                users.push_back(std::move(user.value()));
            }
            return Result<std::vector<User>>::ok(std::move(users));
        } catch (const std::exception& e) {
            return Result<std::vector<User>>::err(
                "PgUserRepository::findAll: " + std::string(e.what())
            );
        }
    }

    // -------------------------------------------------------------------------
    // save (upsert)
    // -------------------------------------------------------------------------
    Result<void> save(const User& user) override {
        try {
            pqxx::work tx{*conn_};
            tx.exec_params(
                "INSERT INTO users (id, name, email, password_hash) "
                "VALUES ($1, $2, $3, $4) "
                "ON CONFLICT (id) DO UPDATE SET "
                "    name          = EXCLUDED.name, "
                "    email         = EXCLUDED.email, "
                "    password_hash = EXCLUDED.password_hash",
                user.id().value(),
                user.name(),
                user.email().value(),
                user.passwordHash()
            );
            tx.commit();
            return Result<void>::ok();
        } catch (const std::exception& e) {
            return Result<void>::err(
                "PgUserRepository::save: " + std::string(e.what())
            );
        }
    }

    // -------------------------------------------------------------------------
    // remove
    // -------------------------------------------------------------------------
    Result<void> remove(const UserId& id) override {
        try {
            pqxx::work tx{*conn_};
            tx.exec_params(
                "DELETE FROM users WHERE id = $1",
                id.value()
            );
            tx.commit();
            return Result<void>::ok();
        } catch (const std::exception& e) {
            return Result<void>::err(
                "PgUserRepository::remove: " + std::string(e.what())
            );
        }
    }

private:
    std::shared_ptr<pqxx::connection> conn_;

    // DB 行 → User エンティティ再構築
    static Result<User> reconstruct(const pqxx::row& row) {
        auto idResult = UserId::create(row["id"].as<std::string>());
        if (!idResult) return Result<User>::err(idResult.error());

        auto emailResult = Email::create(row["email"].as<std::string>());
        if (!emailResult) return Result<User>::err(emailResult.error());

        auto userResult = User::create(
            std::move(idResult.value()),
            row["name"].as<std::string>(),
            std::move(emailResult.value()),
            row["password_hash"].as<std::string>()
        );
        if (!userResult) return userResult;

        userResult.value().setTimestamps(
            row["created_at"].as<std::string>(),
            row["updated_at"].as<std::string>()
        );
        return userResult;
    }
};
