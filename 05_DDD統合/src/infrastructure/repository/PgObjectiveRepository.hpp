#pragma once

#include <memory>
#include <optional>
#include <vector>
#include <stdexcept>

#include <pqxx/pqxx>

#include "../../domain/repository/IObjectiveRepository.hpp"
#include "../../domain/model/objective/Objective.hpp"
#include "../../domain/model/objective/ObjectiveId.hpp"
#include "../../domain/model/period/PeriodId.hpp"
#include "../../domain/model/user/UserId.hpp"
#include "../../common/Result.hpp"

// =============================================================================
// PgObjectiveRepository — IObjectiveRepository の PostgreSQL（libpqxx）実装
//
// インフラ層。pqxx::connection を shared_ptr で受け取る。
//
// SQL 方針:
//   - save(): INSERT ... ON CONFLICT (id) DO UPDATE SET ...（upsert）
//   - findByPeriodId(): display_order ASC でソート
//   - description は NULL 許容（optional<string>）
// =============================================================================
class PgObjectiveRepository : public IObjectiveRepository {
public:
    explicit PgObjectiveRepository(std::shared_ptr<pqxx::connection> conn)
        : conn_(std::move(conn)) {}

    // -------------------------------------------------------------------------
    // findById
    // -------------------------------------------------------------------------
    Result<std::optional<Objective>> findById(const ObjectiveId& id) const override {
        try {
            pqxx::work tx{*conn_};
            pqxx::result rows = tx.exec_params(
                "SELECT id, period_id, owner_id, title, description, "
                "       display_order, created_at::TEXT, updated_at::TEXT "
                "FROM objectives WHERE id = $1",
                id.value()
            );
            tx.commit();

            if (rows.empty()) {
                return Result<std::optional<Objective>>::ok(std::nullopt);
            }
            auto obj = reconstruct(rows[0]);
            if (!obj) return Result<std::optional<Objective>>::err(obj.error());
            return Result<std::optional<Objective>>::ok(
                std::optional<Objective>{std::move(obj.value())}
            );
        } catch (const std::exception& e) {
            return Result<std::optional<Objective>>::err(
                "PgObjectiveRepository::findById: " + std::string(e.what())
            );
        }
    }

    // -------------------------------------------------------------------------
    // findByPeriodId
    // -------------------------------------------------------------------------
    Result<std::vector<Objective>> findByPeriodId(const PeriodId& periodId) const override {
        try {
            pqxx::work tx{*conn_};
            pqxx::result rows = tx.exec_params(
                "SELECT id, period_id, owner_id, title, description, "
                "       display_order, created_at::TEXT, updated_at::TEXT "
                "FROM objectives WHERE period_id = $1 "
                "ORDER BY display_order ASC",
                periodId.value()
            );
            tx.commit();

            return collectObjectives(rows);
        } catch (const std::exception& e) {
            return Result<std::vector<Objective>>::err(
                "PgObjectiveRepository::findByPeriodId: " + std::string(e.what())
            );
        }
    }

    // -------------------------------------------------------------------------
    // findByOwnerId
    // -------------------------------------------------------------------------
    Result<std::vector<Objective>> findByOwnerId(const UserId& ownerId) const override {
        try {
            pqxx::work tx{*conn_};
            pqxx::result rows = tx.exec_params(
                "SELECT id, period_id, owner_id, title, description, "
                "       display_order, created_at::TEXT, updated_at::TEXT "
                "FROM objectives WHERE owner_id = $1 "
                "ORDER BY created_at ASC",
                ownerId.value()
            );
            tx.commit();

            return collectObjectives(rows);
        } catch (const std::exception& e) {
            return Result<std::vector<Objective>>::err(
                "PgObjectiveRepository::findByOwnerId: " + std::string(e.what())
            );
        }
    }

    // -------------------------------------------------------------------------
    // save (upsert)
    // -------------------------------------------------------------------------
    Result<void> save(const Objective& objective) override {
        try {
            pqxx::work tx{*conn_};

            // description は optional<string> なので NULL になりうる
            if (objective.description().has_value()) {
                tx.exec_params(
                    "INSERT INTO objectives "
                    "    (id, period_id, owner_id, title, description, display_order) "
                    "VALUES ($1, $2, $3, $4, $5, $6) "
                    "ON CONFLICT (id) DO UPDATE SET "
                    "    period_id     = EXCLUDED.period_id, "
                    "    owner_id      = EXCLUDED.owner_id, "
                    "    title         = EXCLUDED.title, "
                    "    description   = EXCLUDED.description, "
                    "    display_order = EXCLUDED.display_order",
                    objective.id().value(),
                    objective.periodId().value(),
                    objective.ownerId().value(),
                    objective.title(),
                    objective.description().value(),
                    objective.displayOrder()
                );
            } else {
                tx.exec_params(
                    "INSERT INTO objectives "
                    "    (id, period_id, owner_id, title, description, display_order) "
                    "VALUES ($1, $2, $3, $4, NULL, $5) "
                    "ON CONFLICT (id) DO UPDATE SET "
                    "    period_id     = EXCLUDED.period_id, "
                    "    owner_id      = EXCLUDED.owner_id, "
                    "    title         = EXCLUDED.title, "
                    "    description   = NULL, "
                    "    display_order = EXCLUDED.display_order",
                    objective.id().value(),
                    objective.periodId().value(),
                    objective.ownerId().value(),
                    objective.title(),
                    objective.displayOrder()
                );
            }

            tx.commit();
            return Result<void>::ok();
        } catch (const std::exception& e) {
            return Result<void>::err(
                "PgObjectiveRepository::save: " + std::string(e.what())
            );
        }
    }

    // -------------------------------------------------------------------------
    // remove
    // -------------------------------------------------------------------------
    Result<void> remove(const ObjectiveId& id) override {
        try {
            pqxx::work tx{*conn_};
            tx.exec_params(
                "DELETE FROM objectives WHERE id = $1",
                id.value()
            );
            tx.commit();
            return Result<void>::ok();
        } catch (const std::exception& e) {
            return Result<void>::err(
                "PgObjectiveRepository::remove: " + std::string(e.what())
            );
        }
    }

private:
    std::shared_ptr<pqxx::connection> conn_;

    // DB 行 → Objective エンティティ再構築
    static Result<Objective> reconstruct(const pqxx::row& row) {
        auto idResult = ObjectiveId::create(row["id"].as<std::string>());
        if (!idResult) return Result<Objective>::err(idResult.error());

        auto periodIdResult = PeriodId::create(row["period_id"].as<std::string>());
        if (!periodIdResult) return Result<Objective>::err(periodIdResult.error());

        auto ownerIdResult = UserId::create(row["owner_id"].as<std::string>());
        if (!ownerIdResult) return Result<Objective>::err(ownerIdResult.error());

        std::optional<std::string> description = std::nullopt;
        if (!row["description"].is_null()) {
            description = row["description"].as<std::string>();
        }

        auto objResult = Objective::create(
            std::move(idResult.value()),
            std::move(periodIdResult.value()),
            std::move(ownerIdResult.value()),
            row["title"].as<std::string>(),
            std::move(description),
            row["display_order"].as<int>()
        );
        if (!objResult) return objResult;

        objResult.value().setTimestamps(
            row["created_at"].as<std::string>(),
            row["updated_at"].as<std::string>()
        );
        return objResult;
    }

    // pqxx::result → vector<Objective> への変換（共通処理）
    static Result<std::vector<Objective>> collectObjectives(const pqxx::result& rows) {
        std::vector<Objective> objectives;
        objectives.reserve(rows.size());
        for (const auto& row : rows) {
            auto obj = reconstruct(row);
            if (!obj) return Result<std::vector<Objective>>::err(obj.error());
            objectives.push_back(std::move(obj.value()));
        }
        return Result<std::vector<Objective>>::ok(std::move(objectives));
    }
};
