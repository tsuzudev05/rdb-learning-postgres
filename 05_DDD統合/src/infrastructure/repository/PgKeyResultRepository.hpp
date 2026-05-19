#pragma once

#include <memory>
#include <optional>
#include <vector>
#include <string>
#include <stdexcept>
#include <unordered_map>

#include <pqxx/pqxx>

#include "../../domain/repository/IKeyResultRepository.hpp"
#include "../../domain/model/keyresult/KeyResult.hpp"
#include "../../domain/model/keyresult/KeyResultId.hpp"
#include "../../domain/model/keyresult/KeyResultProgress.hpp"
#include "../../domain/model/keyresult/KrProgressLog.hpp"
#include "../../domain/model/keyresult/KrProgressLogId.hpp"
#include "../../domain/model/objective/ObjectiveId.hpp"
#include "../../domain/model/user/UserId.hpp"
#include "../../common/Result.hpp"

// =============================================================================
// PgKeyResultRepository — IKeyResultRepository の PostgreSQL（libpqxx）実装
//
// インフラ層。pqxx::connection を shared_ptr で受け取る。
//
// KeyResult 集約は KeyResult + KrProgressLog を一括で管理する。
//
// SQL 方針:
//   - findById / findByXxx():
//       key_results と kr_progress_logs を LEFT JOIN して取得し、
//       C++ 側で KeyResult に KrProgressLog を組み込む。
//   - save():
//       1. key_results を upsert（INSERT ... ON CONFLICT (id) DO UPDATE）
//       2. progressLogs_ を INSERT ... ON CONFLICT (id) DO NOTHING で追記
//          （ログは不変：一度保存したら変更しない）
//   - remove():
//       key_results を削除（kr_progress_logs は CASCADE で連動削除）
// =============================================================================
class PgKeyResultRepository : public IKeyResultRepository {
public:
    explicit PgKeyResultRepository(std::shared_ptr<pqxx::connection> conn)
        : conn_(std::move(conn)) {}

    // -------------------------------------------------------------------------
    // findById
    // -------------------------------------------------------------------------
    Result<std::optional<KeyResult>> findById(const KeyResultId& id) const override {
        try {
            pqxx::work tx{*conn_};
            auto rows = fetchWithLogs(tx,
                "WHERE kr.id = $1",
                id.value()
            );
            tx.commit();

            if (rows.empty()) {
                return Result<std::optional<KeyResult>>::ok(std::nullopt);
            }
            auto kr = assembleSingle(rows);
            if (!kr) return Result<std::optional<KeyResult>>::err(kr.error());
            return Result<std::optional<KeyResult>>::ok(
                std::optional<KeyResult>{std::move(kr.value())}
            );
        } catch (const std::exception& e) {
            return Result<std::optional<KeyResult>>::err(
                "PgKeyResultRepository::findById: " + std::string(e.what())
            );
        }
    }

    // -------------------------------------------------------------------------
    // findByObjectiveId
    // -------------------------------------------------------------------------
    Result<std::vector<KeyResult>> findByObjectiveId(
        const ObjectiveId& objectiveId
    ) const override {
        try {
            pqxx::work tx{*conn_};
            auto rows = fetchWithLogs(tx,
                "WHERE kr.objective_id = $1 ORDER BY kr.display_order ASC, log.recorded_at ASC",
                objectiveId.value()
            );
            tx.commit();

            return assembleMany(rows);
        } catch (const std::exception& e) {
            return Result<std::vector<KeyResult>>::err(
                "PgKeyResultRepository::findByObjectiveId: " + std::string(e.what())
            );
        }
    }

    // -------------------------------------------------------------------------
    // findByOwnerId
    // -------------------------------------------------------------------------
    Result<std::vector<KeyResult>> findByOwnerId(const UserId& ownerId) const override {
        try {
            pqxx::work tx{*conn_};
            auto rows = fetchWithLogs(tx,
                "WHERE kr.owner_id = $1 ORDER BY kr.created_at ASC, log.recorded_at ASC",
                ownerId.value()
            );
            tx.commit();

            return assembleMany(rows);
        } catch (const std::exception& e) {
            return Result<std::vector<KeyResult>>::err(
                "PgKeyResultRepository::findByOwnerId: " + std::string(e.what())
            );
        }
    }

    // -------------------------------------------------------------------------
    // save (upsert)
    // KeyResult を upsert したあと、progressLogs_ を追記（ON CONFLICT DO NOTHING）
    // -------------------------------------------------------------------------
    Result<void> save(const KeyResult& kr) override {
        try {
            pqxx::work tx{*conn_};

            // 1. key_results upsert
            if (std::holds_alternative<NumericProgress>(kr.progress())) {
                const auto& np = std::get<NumericProgress>(kr.progress());
                tx.exec_params(
                    "INSERT INTO key_results "
                    "    (id, objective_id, owner_id, title, kr_type, "
                    "     target_value, current_value, is_completed, display_order) "
                    "VALUES ($1, $2, $3, $4, 'numeric', $5, $6, FALSE, $7) "
                    "ON CONFLICT (id) DO UPDATE SET "
                    "    objective_id  = EXCLUDED.objective_id, "
                    "    owner_id      = EXCLUDED.owner_id, "
                    "    title         = EXCLUDED.title, "
                    "    target_value  = EXCLUDED.target_value, "
                    "    current_value = EXCLUDED.current_value, "
                    "    display_order = EXCLUDED.display_order",
                    kr.id().value(),
                    kr.objectiveId().value(),
                    kr.ownerId().value(),
                    kr.title(),
                    np.targetValue(),
                    np.currentValue(),
                    kr.displayOrder()
                );
            } else {
                const auto& cp = std::get<CheckboxProgress>(kr.progress());
                tx.exec_params(
                    "INSERT INTO key_results "
                    "    (id, objective_id, owner_id, title, kr_type, "
                    "     target_value, current_value, is_completed, display_order) "
                    "VALUES ($1, $2, $3, $4, 'checkbox', NULL, 0, $5, $6) "
                    "ON CONFLICT (id) DO UPDATE SET "
                    "    objective_id  = EXCLUDED.objective_id, "
                    "    owner_id      = EXCLUDED.owner_id, "
                    "    title         = EXCLUDED.title, "
                    "    is_completed  = EXCLUDED.is_completed, "
                    "    display_order = EXCLUDED.display_order",
                    kr.id().value(),
                    kr.objectiveId().value(),
                    kr.ownerId().value(),
                    kr.title(),
                    cp.isCompleted(),
                    kr.displayOrder()
                );
            }

            // 2. KrProgressLog を追記（既存ログは変更しない）
            for (const auto& log : kr.progressLogs()) {
                saveProgressLog(tx, log);
            }

            tx.commit();
            return Result<void>::ok();
        } catch (const std::exception& e) {
            return Result<void>::err(
                "PgKeyResultRepository::save: " + std::string(e.what())
            );
        }
    }

    // -------------------------------------------------------------------------
    // remove
    // kr_progress_logs は ON DELETE CASCADE で自動削除される
    // -------------------------------------------------------------------------
    Result<void> remove(const KeyResultId& id) override {
        try {
            pqxx::work tx{*conn_};
            tx.exec_params(
                "DELETE FROM key_results WHERE id = $1",
                id.value()
            );
            tx.commit();
            return Result<void>::ok();
        } catch (const std::exception& e) {
            return Result<void>::err(
                "PgKeyResultRepository::remove: " + std::string(e.what())
            );
        }
    }

private:
    std::shared_ptr<pqxx::connection> conn_;

    // -------------------------------------------------------------------------
    // SQL ヘルパー：key_results LEFT JOIN kr_progress_logs で一括取得
    // -------------------------------------------------------------------------
    pqxx::result fetchWithLogs(
        pqxx::work& tx,
        const std::string& whereClause,
        const std::string& param
    ) const {
        const std::string sql =
            "SELECT "
            "    kr.id              AS kr_id, "
            "    kr.objective_id    AS kr_objective_id, "
            "    kr.owner_id        AS kr_owner_id, "
            "    kr.title           AS kr_title, "
            "    kr.kr_type         AS kr_type, "
            "    kr.target_value    AS kr_target_value, "
            "    kr.current_value   AS kr_current_value, "
            "    kr.is_completed    AS kr_is_completed, "
            "    kr.display_order   AS kr_display_order, "
            "    kr.created_at::TEXT AS kr_created_at, "
            "    kr.updated_at::TEXT AS kr_updated_at, "
            "    log.id             AS log_id, "
            "    log.recorded_by    AS log_recorded_by, "
            "    log.value          AS log_value, "
            "    log.completed      AS log_completed, "
            "    log.note           AS log_note, "
            "    log.recorded_at::TEXT AS log_recorded_at "
            "FROM key_results kr "
            "LEFT JOIN kr_progress_logs log ON log.key_result_id = kr.id "
            + whereClause;
        return tx.exec_params(sql, param);
    }

    // -------------------------------------------------------------------------
    // 行群から KeyResult 1件を組み立てる（findById 用）
    // -------------------------------------------------------------------------
    static Result<KeyResult> assembleSingle(const pqxx::result& rows) {
        // すべての行は同じ kr_id を持つ
        auto krResult = reconstructKeyResult(rows[0]);
        if (!krResult) return krResult;

        std::vector<KrProgressLog> logs;
        for (const auto& row : rows) {
            if (row["log_id"].is_null()) continue;  // LEFT JOIN で log なし
            auto logResult = reconstructLog(row);
            if (!logResult) return Result<KeyResult>::err(logResult.error());
            logs.push_back(std::move(logResult.value()));
        }
        krResult.value().setProgressLogs(std::move(logs));
        return krResult;
    }

    // -------------------------------------------------------------------------
    // 行群から KeyResult 複数件を組み立てる（findByXxx 用）
    // kr_id でグルーピングしながら順序を保持する
    // -------------------------------------------------------------------------
    static Result<std::vector<KeyResult>> assembleMany(const pqxx::result& rows) {
        // 出現順を保持するため、ID の出現順リストと map を併用する
        std::vector<std::string> order;
        std::unordered_map<std::string, pqxx::result::size_type> firstRowIdx;
        std::unordered_map<std::string, std::vector<std::size_t>> logRowIdxMap;

        for (pqxx::result::size_type i = 0; i < rows.size(); ++i) {
            const std::string krId = rows[i]["kr_id"].as<std::string>();
            if (firstRowIdx.find(krId) == firstRowIdx.end()) {
                firstRowIdx[krId] = i;
                order.push_back(krId);
            }
            if (!rows[i]["log_id"].is_null()) {
                logRowIdxMap[krId].push_back(static_cast<std::size_t>(i));
            }
        }

        std::vector<KeyResult> result;
        result.reserve(order.size());

        for (const auto& krId : order) {
            auto krResult = reconstructKeyResult(rows[firstRowIdx[krId]]);
            if (!krResult) return Result<std::vector<KeyResult>>::err(krResult.error());

            std::vector<KrProgressLog> logs;
            if (logRowIdxMap.count(krId)) {
                for (auto rowIdx : logRowIdxMap[krId]) {
                    auto logResult = reconstructLog(rows[rowIdx]);
                    if (!logResult) {
                        return Result<std::vector<KeyResult>>::err(logResult.error());
                    }
                    logs.push_back(std::move(logResult.value()));
                }
            }
            krResult.value().setProgressLogs(std::move(logs));
            result.push_back(std::move(krResult.value()));
        }
        return Result<std::vector<KeyResult>>::ok(std::move(result));
    }

    // -------------------------------------------------------------------------
    // DB 行 → KeyResult エンティティ再構築（progress_logs は別途セット）
    // -------------------------------------------------------------------------
    static Result<KeyResult> reconstructKeyResult(const pqxx::row& row) {
        auto idResult = KeyResultId::create(row["kr_id"].as<std::string>());
        if (!idResult) return Result<KeyResult>::err(idResult.error());

        auto objectiveIdResult = ObjectiveId::create(row["kr_objective_id"].as<std::string>());
        if (!objectiveIdResult) return Result<KeyResult>::err(objectiveIdResult.error());

        auto ownerIdResult = UserId::create(row["kr_owner_id"].as<std::string>());
        if (!ownerIdResult) return Result<KeyResult>::err(ownerIdResult.error());

        // KeyResultProgress の再構築
        const std::string krType = row["kr_type"].as<std::string>();
        KeyResultProgress progress = [&]() -> KeyResultProgress {
            if (krType == "numeric") {
                double target  = row["kr_target_value"].as<double>();
                double current = row["kr_current_value"].as<double>();
                auto np = NumericProgress::create(target, current);
                // DB の CHECK 制約で保証済みのため、エラーは想定しない
                return np.value();
            } else {
                bool completed = row["kr_is_completed"].as<bool>();
                return CheckboxProgress::create(completed);
            }
        }();

        auto krResult = KeyResult::create(
            std::move(idResult.value()),
            std::move(objectiveIdResult.value()),
            std::move(ownerIdResult.value()),
            row["kr_title"].as<std::string>(),
            std::move(progress),
            row["kr_display_order"].as<int>()
        );
        if (!krResult) return krResult;

        krResult.value().setTimestamps(
            row["kr_created_at"].as<std::string>(),
            row["kr_updated_at"].as<std::string>()
        );
        return krResult;
    }

    // -------------------------------------------------------------------------
    // DB 行 → KrProgressLog エンティティ再構築
    // -------------------------------------------------------------------------
    static Result<KrProgressLog> reconstructLog(const pqxx::row& row) {
        auto logIdResult = KrProgressLogId::create(row["log_id"].as<std::string>());
        if (!logIdResult) return Result<KrProgressLog>::err(logIdResult.error());

        // kr_id は外部結合元から取得
        auto krIdResult = KeyResultId::create(row["kr_id"].as<std::string>());
        if (!krIdResult) return Result<KrProgressLog>::err(krIdResult.error());

        auto recordedByResult = UserId::create(row["log_recorded_by"].as<std::string>());
        if (!recordedByResult) return Result<KrProgressLog>::err(recordedByResult.error());

        std::optional<std::string> note = std::nullopt;
        if (!row["log_note"].is_null()) {
            note = row["log_note"].as<std::string>();
        }
        const std::string recordedAt = row["log_recorded_at"].as<std::string>();

        if (!row["log_value"].is_null()) {
            // numeric ログ
            return KrProgressLog::createNumeric(
                std::move(logIdResult.value()),
                std::move(krIdResult.value()),
                std::move(recordedByResult.value()),
                row["log_value"].as<double>(),
                std::move(note),
                recordedAt
            );
        } else {
            // checkbox ログ
            return KrProgressLog::createCheckbox(
                std::move(logIdResult.value()),
                std::move(krIdResult.value()),
                std::move(recordedByResult.value()),
                row["log_completed"].as<bool>(),
                std::move(note),
                recordedAt
            );
        }
    }

    // -------------------------------------------------------------------------
    // KrProgressLog を DB に追記（既存ログは変更しない）
    //
    // libpqxx 7.x は std::optional<T> をパラメータとして受け付ける。
    // nullopt を渡すと SQL の NULL として扱われる。
    // recorded_at が空文字の場合は DEFAULT（NOW()）に任せるため列を省く SQL を使う。
    // -------------------------------------------------------------------------
    static void saveProgressLog(pqxx::work& tx, const KrProgressLog& log) {
        // note は optional<string> をそのまま渡す（nullopt → NULL）
        const std::optional<std::string>& note = log.note();

        if (log.isNumeric()) {
            if (log.recordedAt().empty()) {
                tx.exec_params(
                    "INSERT INTO kr_progress_logs "
                    "    (id, key_result_id, recorded_by, value, completed, note) "
                    "VALUES ($1, $2, $3, $4, NULL, $5) "
                    "ON CONFLICT (id) DO NOTHING",
                    log.id().value(),
                    log.keyResultId().value(),
                    log.recordedBy().value(),
                    log.value().value(),   // double
                    note                   // optional<string> → NULL if nullopt
                );
            } else {
                tx.exec_params(
                    "INSERT INTO kr_progress_logs "
                    "    (id, key_result_id, recorded_by, value, completed, note, recorded_at) "
                    "VALUES ($1, $2, $3, $4, NULL, $5, $6::TIMESTAMPTZ) "
                    "ON CONFLICT (id) DO NOTHING",
                    log.id().value(),
                    log.keyResultId().value(),
                    log.recordedBy().value(),
                    log.value().value(),
                    note,
                    log.recordedAt()
                );
            }
        } else {
            if (log.recordedAt().empty()) {
                tx.exec_params(
                    "INSERT INTO kr_progress_logs "
                    "    (id, key_result_id, recorded_by, value, completed, note) "
                    "VALUES ($1, $2, $3, NULL, $4, $5) "
                    "ON CONFLICT (id) DO NOTHING",
                    log.id().value(),
                    log.keyResultId().value(),
                    log.recordedBy().value(),
                    log.completed().value(),  // bool
                    note
                );
            } else {
                tx.exec_params(
                    "INSERT INTO kr_progress_logs "
                    "    (id, key_result_id, recorded_by, value, completed, note, recorded_at) "
                    "VALUES ($1, $2, $3, NULL, $4, $5, $6::TIMESTAMPTZ) "
                    "ON CONFLICT (id) DO NOTHING",
                    log.id().value(),
                    log.keyResultId().value(),
                    log.recordedBy().value(),
                    log.completed().value(),
                    note,
                    log.recordedAt()
                );
            }
        }
    }
};
