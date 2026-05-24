#pragma once

#include <memory>
#include <optional>
#include <vector>
#include <stdexcept>
#include <map>

#include <pqxx/pqxx>

#include "../../domain/repository/ITeamRepository.hpp"
#include "../../domain/model/team/Team.hpp"
#include "../../domain/model/team/TeamId.hpp"
#include "../../domain/model/team/TeamMember.hpp"
#include "../../domain/model/team/TeamMemberId.hpp"
#include "../../domain/model/team/Role.hpp"
#include "../../domain/model/user/UserId.hpp"
#include "../../common/Result.hpp"

// =============================================================================
// PgTeamRepository — ITeamRepository の PostgreSQL（libpqxx）実装
//
// インフラ層。pqxx::connection を shared_ptr で受け取り、
// 各メソッドでトランザクションを開いて操作する。
//
// SQL 方針:
//   - save(): teams を upsert → team_members を DELETE & INSERT で一括置換
//             （Team 集約の整合性を保つため全削除→再挿入）
//   - remove(): teams を DELETE（team_members は ON DELETE CASCADE）
//   - findById/findByUserId/findAll: teams と team_members を LEFT JOIN で取得
// =============================================================================
class PgTeamRepository : public ITeamRepository {
public:
    explicit PgTeamRepository(std::shared_ptr<pqxx::connection> conn)
        : conn_(std::move(conn)) {}

    // -------------------------------------------------------------------------
    // findById
    // -------------------------------------------------------------------------
    Result<std::optional<Team>> findById(const TeamId& id) const override {
        try {
            pqxx::work tx{*conn_};
            pqxx::result rows = tx.exec_params(
                "SELECT t.id, t.name, t.description, "
                "       t.created_at::TEXT, t.updated_at::TEXT, "
                "       tm.id AS member_id, tm.user_id, tm.role, tm.joined_at::TEXT "
                "FROM teams t "
                "LEFT JOIN team_members tm ON t.id = tm.team_id "
                "WHERE t.id = $1 "
                "ORDER BY tm.joined_at ASC",
                id.value()
            );
            tx.commit();

            if (rows.empty()) {
                return Result<std::optional<Team>>::ok(std::nullopt);
            }

            auto teamResult = reconstructTeamWithMembers(rows);
            if (!teamResult) return Result<std::optional<Team>>::err(teamResult.error());
            return Result<std::optional<Team>>::ok(
                std::optional<Team>{std::move(teamResult.value())}
            );
        } catch (const std::exception& e) {
            return Result<std::optional<Team>>::err(
                "PgTeamRepository::findById: " + std::string(e.what())
            );
        }
    }

    // -------------------------------------------------------------------------
    // findByUserId
    // -------------------------------------------------------------------------
    Result<std::vector<Team>> findByUserId(const UserId& userId) const override {
        try {
            pqxx::work tx{*conn_};
            // まず対象チームのIDを取得
            pqxx::result teamIdRows = tx.exec_params(
                "SELECT DISTINCT t.id "
                "FROM teams t "
                "JOIN team_members tm ON t.id = tm.team_id "
                "WHERE tm.user_id = $1",
                userId.value()
            );

            if (teamIdRows.empty()) {
                tx.commit();
                return Result<std::vector<Team>>::ok({});
            }

            // チームIDリストを IN 句用に構築
            std::string idList;
            for (pqxx::result::size_type i = 0; i < teamIdRows.size(); ++i) {
                if (i > 0) idList += ", ";
                idList += tx.quote(teamIdRows[i][0].as<std::string>());
            }

            // 全メンバー含めて取得
            pqxx::result rows = tx.exec(
                "SELECT t.id, t.name, t.description, "
                "       t.created_at::TEXT, t.updated_at::TEXT, "
                "       tm.id AS member_id, tm.user_id, tm.role, tm.joined_at::TEXT "
                "FROM teams t "
                "LEFT JOIN team_members tm ON t.id = tm.team_id "
                "WHERE t.id IN (" + idList + ") "
                "ORDER BY t.created_at ASC, tm.joined_at ASC"
            );
            tx.commit();

            return reconstructTeamList(rows);
        } catch (const std::exception& e) {
            return Result<std::vector<Team>>::err(
                "PgTeamRepository::findByUserId: " + std::string(e.what())
            );
        }
    }

    // -------------------------------------------------------------------------
    // findAll
    // -------------------------------------------------------------------------
    Result<std::vector<Team>> findAll() const override {
        try {
            pqxx::work tx{*conn_};
            pqxx::result rows = tx.exec(
                "SELECT t.id, t.name, t.description, "
                "       t.created_at::TEXT, t.updated_at::TEXT, "
                "       tm.id AS member_id, tm.user_id, tm.role, tm.joined_at::TEXT "
                "FROM teams t "
                "LEFT JOIN team_members tm ON t.id = tm.team_id "
                "ORDER BY t.created_at ASC, tm.joined_at ASC"
            );
            tx.commit();

            return reconstructTeamList(rows);
        } catch (const std::exception& e) {
            return Result<std::vector<Team>>::err(
                "PgTeamRepository::findAll: " + std::string(e.what())
            );
        }
    }

    // -------------------------------------------------------------------------
    // save (upsert)
    // Team + TeamMember を一括保存する。
    // team_members は「全削除 → 再挿入」で集約整合性を保証する。
    // -------------------------------------------------------------------------
    Result<void> save(const Team& team) override {
        try {
            pqxx::work tx{*conn_};

            // 1. teams テーブルを upsert
            tx.exec_params(
                "INSERT INTO teams (id, name, description) "
                "VALUES ($1, $2, $3) "
                "ON CONFLICT (id) DO UPDATE SET "
                "    name        = EXCLUDED.name, "
                "    description = EXCLUDED.description",
                team.id().value(),
                team.name(),
                team.description().has_value()
                    ? std::optional<std::string>{team.description().value()}
                    : std::optional<std::string>{std::nullopt}
            );

            // 2. 既存の team_members を全削除
            tx.exec_params(
                "DELETE FROM team_members WHERE team_id = $1",
                team.id().value()
            );

            // 3. 現在のメンバーを再挿入
            for (const auto& member : team.members()) {
                tx.exec_params(
                    "INSERT INTO team_members (id, team_id, user_id, role) "
                    "VALUES ($1, $2, $3, $4)",
                    member.id().value(),
                    member.teamId().value(),
                    member.userId().value(),
                    member.role().toString()
                );
            }

            tx.commit();
            return Result<void>::ok();
        } catch (const std::exception& e) {
            return Result<void>::err(
                "PgTeamRepository::save: " + std::string(e.what())
            );
        }
    }

    // -------------------------------------------------------------------------
    // remove
    // -------------------------------------------------------------------------
    Result<void> remove(const TeamId& id) override {
        try {
            pqxx::work tx{*conn_};
            tx.exec_params(
                "DELETE FROM teams WHERE id = $1",
                id.value()
            );
            tx.commit();
            return Result<void>::ok();
        } catch (const std::exception& e) {
            return Result<void>::err(
                "PgTeamRepository::remove: " + std::string(e.what())
            );
        }
    }

private:
    std::shared_ptr<pqxx::connection> conn_;

    // -------------------------------------------------------------------------
    // 内部ユーティリティ
    // -------------------------------------------------------------------------

    // DB行1グループ（同一チームの全行）から Team を再構築する
    // rows は同一チームの行が連続していること前提（WHERE id = $1 の場合）
    static Result<Team> reconstructTeamWithMembers(const pqxx::result& rows) {
        const auto& firstRow = rows[0];

        auto idResult = TeamId::create(firstRow["id"].as<std::string>());
        if (!idResult) return Result<Team>::err(idResult.error());

        auto teamResult = Team::create(
            std::move(idResult.value()),
            firstRow["name"].as<std::string>(),
            firstRow["description"].is_null()
                ? std::optional<std::string>{std::nullopt}
                : std::optional<std::string>{firstRow["description"].as<std::string>()}
        );
        if (!teamResult) return teamResult;
        Team& team = teamResult.value();

        team.setTimestamps(
            firstRow["created_at"].as<std::string>(),
            firstRow["updated_at"].as<std::string>()
        );

        // メンバーを追加
        std::vector<TeamMember> members;
        for (const auto& row : rows) {
            if (row["member_id"].is_null()) continue; // メンバーなし（LEFT JOIN の NULL）

            auto memberIdResult = TeamMemberId::create(row["member_id"].as<std::string>());
            if (!memberIdResult) return Result<Team>::err(memberIdResult.error());

            auto memberTeamIdResult = TeamId::create(firstRow["id"].as<std::string>());
            if (!memberTeamIdResult) return Result<Team>::err(memberTeamIdResult.error());

            auto userIdResult = UserId::create(row["user_id"].as<std::string>());
            if (!userIdResult) return Result<Team>::err(userIdResult.error());

            auto roleResult = Role::create(row["role"].as<std::string>());
            if (!roleResult) return Result<Team>::err(roleResult.error());

            auto memberResult = TeamMember::create(
                std::move(memberIdResult.value()),
                std::move(memberTeamIdResult.value()),
                std::move(userIdResult.value()),
                std::move(roleResult.value()),
                row["joined_at"].is_null() ? "" : row["joined_at"].as<std::string>()
            );
            if (!memberResult) return Result<Team>::err(memberResult.error());
            members.push_back(std::move(memberResult.value()));
        }
        team.setMembers(std::move(members));

        return Result<Team>::ok(std::move(team));
    }

    // 複数チームの rows（ORDER BY t.id, tm.joined_at）から Team のリストを再構築する
    static Result<std::vector<Team>> reconstructTeamList(const pqxx::result& rows) {
        std::vector<Team> teams;
        if (rows.empty()) return Result<std::vector<Team>>::ok(teams);

        // team_id でグループ化
        std::string currentId;
        std::vector<pqxx::row> currentGroup;

        auto flush = [&]() -> Result<void> {
            if (currentGroup.empty()) return Result<void>::ok();

            // pqxx::result を直接使えないので手動でグループをサブクエリ的に処理
            // reconstructTeamWithMembers は pqxx::result を要求するため、
            // ここでは直接 Team を組み立てる
            const auto& firstRow = currentGroup[0];

            auto idResult = TeamId::create(firstRow["id"].as<std::string>());
            if (!idResult) return Result<void>::err(idResult.error());

            auto teamResult = Team::create(
                std::move(idResult.value()),
                firstRow["name"].as<std::string>(),
                firstRow["description"].is_null()
                    ? std::optional<std::string>{std::nullopt}
                    : std::optional<std::string>{firstRow["description"].as<std::string>()}
            );
            if (!teamResult) return Result<void>::err(teamResult.error());
            Team& team = teamResult.value();

            team.setTimestamps(
                firstRow["created_at"].as<std::string>(),
                firstRow["updated_at"].as<std::string>()
            );

            std::vector<TeamMember> members;
            for (const auto& row : currentGroup) {
                if (row["member_id"].is_null()) continue;

                auto memberIdResult   = TeamMemberId::create(row["member_id"].as<std::string>());
                if (!memberIdResult) return Result<void>::err(memberIdResult.error());

                auto memberTeamIdResult = TeamId::create(row["id"].as<std::string>());
                if (!memberTeamIdResult) return Result<void>::err(memberTeamIdResult.error());

                auto userIdResult = UserId::create(row["user_id"].as<std::string>());
                if (!userIdResult) return Result<void>::err(userIdResult.error());

                auto roleResult = Role::create(row["role"].as<std::string>());
                if (!roleResult) return Result<void>::err(roleResult.error());

                auto memberResult = TeamMember::create(
                    std::move(memberIdResult.value()),
                    std::move(memberTeamIdResult.value()),
                    std::move(userIdResult.value()),
                    std::move(roleResult.value()),
                    row["joined_at"].is_null() ? "" : row["joined_at"].as<std::string>()
                );
                if (!memberResult) return Result<void>::err(memberResult.error());
                members.push_back(std::move(memberResult.value()));
            }
            team.setMembers(std::move(members));
            teams.push_back(std::move(team));
            currentGroup.clear();
            return Result<void>::ok();
        };

        for (const auto& row : rows) {
            std::string teamId = row["id"].as<std::string>();
            if (teamId != currentId) {
                auto r = flush();
                if (!r) return Result<std::vector<Team>>::err(r.error());
                currentId = teamId;
            }
            currentGroup.push_back(row);
        }
        auto r = flush();
        if (!r) return Result<std::vector<Team>>::err(r.error());

        return Result<std::vector<Team>>::ok(std::move(teams));
    }
};
