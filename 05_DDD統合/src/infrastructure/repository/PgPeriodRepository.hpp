#pragma once

#include <memory>
#include <optional>
#include <vector>
#include <stdexcept>

#include <pqxx/pqxx>

#include "../../domain/repository/IPeriodRepository.hpp"
#include "../../domain/model/period/Period.hpp"
#include "../../domain/model/period/PeriodId.hpp"
#include "../../domain/model/period/Half.hpp"
#include "../../domain/model/period/DateRange.hpp"
#include "../../domain/model/team/TeamId.hpp"
#include "../../common/Result.hpp"

// =============================================================================
// PgPeriodRepository — IPeriodRepository の PostgreSQL（libpqxx）実装
//
// インフラ層。pqxx::connection を shared_ptr で受け取り、
// 各メソッドでトランザクションを開いて操作する。
//
// SQL 方針:
//   - save(): INSERT ... ON CONFLICT (id) DO UPDATE SET ... (upsert)
//   - remove(): 対象が存在しなくてもエラーにしない
//   - start_date / end_date は DATE 型 → TEXT でやり取り（DateRange は文字列保持）
// =============================================================================
class PgPeriodRepository : public IPeriodRepository {
public:
    explicit PgPeriodRepository(std::shared_ptr<pqxx::connection> conn)
        : conn_(std::move(conn)) {}

    // -------------------------------------------------------------------------
    // findById
    // -------------------------------------------------------------------------
    Result<std::optional<Period>> findById(const PeriodId& id) const override {
        try {
            pqxx::work tx{*conn_};
            pqxx::result rows = tx.exec_params(
                "SELECT id, team_id, name, half, "
                "       start_date::TEXT, end_date::TEXT, "
                "       created_at::TEXT, updated_at::TEXT "
                "FROM periods WHERE id = $1",
                id.value()
            );
            tx.commit();

            if (rows.empty()) {
                return Result<std::optional<Period>>::ok(std::nullopt);
            }
            auto period = reconstruct(rows[0]);
            if (!period) return Result<std::optional<Period>>::err(period.error());
            return Result<std::optional<Period>>::ok(
                std::optional<Period>{std::move(period.value())}
            );
        } catch (const std::exception& e) {
            return Result<std::optional<Period>>::err(
                "PgPeriodRepository::findById: " + std::string(e.what())
            );
        }
    }

    // -------------------------------------------------------------------------
    // findByTeamId
    // -------------------------------------------------------------------------
    Result<std::vector<Period>> findByTeamId(const TeamId& teamId) const override {
        try {
            pqxx::work tx{*conn_};
            pqxx::result rows = tx.exec_params(
                "SELECT id, team_id, name, half, "
                "       start_date::TEXT, end_date::TEXT, "
                "       created_at::TEXT, updated_at::TEXT "
                "FROM periods WHERE team_id = $1 "
                "ORDER BY start_date ASC",
                teamId.value()
            );
            tx.commit();

            return reconstructList(rows);
        } catch (const std::exception& e) {
            return Result<std::vector<Period>>::err(
                "PgPeriodRepository::findByTeamId: " + std::string(e.what())
            );
        }
    }

    // -------------------------------------------------------------------------
    // findAll
    // -------------------------------------------------------------------------
    Result<std::vector<Period>> findAll() const override {
        try {
            pqxx::work tx{*conn_};
            pqxx::result rows = tx.exec(
                "SELECT id, team_id, name, half, "
                "       start_date::TEXT, end_date::TEXT, "
                "       created_at::TEXT, updated_at::TEXT "
                "FROM periods ORDER BY start_date ASC"
            );
            tx.commit();

            return reconstructList(rows);
        } catch (const std::exception& e) {
            return Result<std::vector<Period>>::err(
                "PgPeriodRepository::findAll: " + std::string(e.what())
            );
        }
    }

    // -------------------------------------------------------------------------
    // save (upsert)
    // -------------------------------------------------------------------------
    Result<void> save(const Period& period) override {
        try {
            pqxx::work tx{*conn_};
            tx.exec_params(
                "INSERT INTO periods (id, team_id, name, half, start_date, end_date) "
                "VALUES ($1, $2, $3, $4, $5::DATE, $6::DATE) "
                "ON CONFLICT (id) DO UPDATE SET "
                "    team_id    = EXCLUDED.team_id, "
                "    name       = EXCLUDED.name, "
                "    half       = EXCLUDED.half, "
                "    start_date = EXCLUDED.start_date, "
                "    end_date   = EXCLUDED.end_date",
                period.id().value(),
                period.teamId().value(),
                period.name(),
                period.half().toString(),
                period.dateRange().startDate(),
                period.dateRange().endDate()
            );
            tx.commit();
            return Result<void>::ok();
        } catch (const std::exception& e) {
            return Result<void>::err(
                "PgPeriodRepository::save: " + std::string(e.what())
            );
        }
    }

    // -------------------------------------------------------------------------
    // remove
    // -------------------------------------------------------------------------
    Result<void> remove(const PeriodId& id) override {
        try {
            pqxx::work tx{*conn_};
            tx.exec_params(
                "DELETE FROM periods WHERE id = $1",
                id.value()
            );
            tx.commit();
            return Result<void>::ok();
        } catch (const std::exception& e) {
            return Result<void>::err(
                "PgPeriodRepository::remove: " + std::string(e.what())
            );
        }
    }

private:
    std::shared_ptr<pqxx::connection> conn_;

    // DB 行 → Period エンティティ再構築
    static Result<Period> reconstruct(const pqxx::row& row) {
        auto idResult = PeriodId::create(row["id"].as<std::string>());
        if (!idResult) return Result<Period>::err(idResult.error());

        auto teamIdResult = TeamId::create(row["team_id"].as<std::string>());
        if (!teamIdResult) return Result<Period>::err(teamIdResult.error());

        auto halfResult = Half::create(row["half"].as<std::string>());
        if (!halfResult) return Result<Period>::err(halfResult.error());

        auto dateRangeResult = DateRange::create(
            row["start_date"].as<std::string>(),
            row["end_date"].as<std::string>()
        );
        if (!dateRangeResult) return Result<Period>::err(dateRangeResult.error());

        auto periodResult = Period::create(
            std::move(idResult.value()),
            std::move(teamIdResult.value()),
            row["name"].as<std::string>(),
            std::move(halfResult.value()),
            std::move(dateRangeResult.value())
        );
        if (!periodResult) return periodResult;

        periodResult.value().setTimestamps(
            row["created_at"].as<std::string>(),
            row["updated_at"].as<std::string>()
        );
        return periodResult;
    }

    // 複数行から Period のリストを再構築する
    static Result<std::vector<Period>> reconstructList(const pqxx::result& rows) {
        std::vector<Period> periods;
        periods.reserve(rows.size());
        for (const auto& row : rows) {
            auto period = reconstruct(row);
            if (!period) return Result<std::vector<Period>>::err(period.error());
            periods.push_back(std::move(period.value()));
        }
        return Result<std::vector<Period>>::ok(std::move(periods));
    }
};
