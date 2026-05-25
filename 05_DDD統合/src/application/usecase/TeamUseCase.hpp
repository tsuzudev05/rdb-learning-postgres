#pragma once

#include <memory>
#include <string>
#include <optional>
#include <vector>
#include "../../domain/model/team/Team.hpp"
#include "../../domain/model/team/TeamId.hpp"
#include "../../domain/model/team/TeamMember.hpp"
#include "../../domain/model/team/TeamMemberId.hpp"
#include "../../domain/model/team/Role.hpp"
#include "../../domain/model/user/UserId.hpp"
#include "../../domain/repository/ITeamRepository.hpp"
#include "../../common/Result.hpp"

// =============================================================================
// TeamUseCase — チームユースケース（アプリケーション層）
//
// ドメイン層の ITeamRepository に依存注入（DI）パターンで依存する。
// インフラ層（libpqxx）への直接依存を持たない。
//
// 提供するユースケース:
//   - CreateTeam        : 新規チームを作成する
//   - GetTeam           : ID でチームを1件取得する（メンバー含む）
//   - ListTeams         : 全チーム一覧を取得する
//   - ListTeamsByUser   : 指定ユーザーが所属するチーム一覧を取得する
//   - DeleteTeam        : ID でチームを削除する
//   - AddMember         : チームにメンバーを追加する
//   - RemoveMember      : チームからメンバーを削除する
//   - ChangeMemberRole  : チームメンバーのロールを変更する
//
// エラーハンドリング:
//   すべての操作は Result<T> を返す。例外は使わない。
// =============================================================================

// ─── 入力 DTO ──────────────────────────────────────────────────────────────

struct CreateTeamInput {
    std::string name;
    std::optional<std::string> description;  // 省略可
};

struct GetTeamInput {
    std::string teamId;  // UUID 文字列
};

struct DeleteTeamInput {
    std::string teamId;  // UUID 文字列
};

struct AddMemberInput {
    std::string teamId;   // UUID 文字列
    std::string userId;   // UUID 文字列
    std::string role;     // "admin" or "member"
};

struct RemoveMemberInput {
    std::string teamId;   // UUID 文字列
    std::string userId;   // UUID 文字列
};

struct ChangeMemberRoleInput {
    std::string teamId;   // UUID 文字列
    std::string userId;   // UUID 文字列
    std::string role;     // "admin" or "member"
};

struct ListTeamsByUserInput {
    std::string userId;   // UUID 文字列
};

// ─── 出力 DTO ──────────────────────────────────────────────────────────────

struct TeamMemberOutput {
    std::string memberId;
    std::string teamId;
    std::string userId;
    std::string role;
    std::string joinedAt;

    static TeamMemberOutput from(const TeamMember& member) {
        return TeamMemberOutput{
            member.id().value(),
            member.teamId().value(),
            member.userId().value(),
            member.role().toString(),
            member.joinedAt()
        };
    }
};

struct TeamOutput {
    std::string id;
    std::string name;
    std::optional<std::string> description;
    std::vector<TeamMemberOutput> members;
    std::string createdAt;
    std::string updatedAt;

    // Team エンティティから変換するファクトリ
    static TeamOutput from(const Team& team) {
        std::vector<TeamMemberOutput> memberOutputs;
        memberOutputs.reserve(team.members().size());
        for (const auto& m : team.members()) {
            memberOutputs.push_back(TeamMemberOutput::from(m));
        }
        return TeamOutput{
            team.id().value(),
            team.name(),
            team.description(),
            std::move(memberOutputs),
            team.createdAt(),
            team.updatedAt()
        };
    }
};

// ─── TeamUseCase ────────────────────────────────────────────────────────────

class TeamUseCase {
public:
    // コンストラクタ：ITeamRepository を DI で受け取る（所有権は共有しない）
    explicit TeamUseCase(std::shared_ptr<ITeamRepository> repo)
        : repo_(std::move(repo)) {}

    // ──────────────────────────────────────────────────────────────────────
    // CreateTeam — 新規チームを作成する
    //
    // 処理フロー:
    //   1. TeamId を新規生成
    //   2. Team エンティティを生成
    //   3. Repository#save で永続化
    // ──────────────────────────────────────────────────────────────────────
    Result<TeamOutput> createTeam(const CreateTeamInput& input) {
        // 1. TeamId を新規生成
        auto idResult = TeamId::generate();
        if (!idResult) {
            return Result<TeamOutput>::err(idResult.error());
        }

        // 2. Team エンティティを生成（バリデーション含む）
        auto teamResult = Team::create(
            idResult.value(),
            input.name,
            input.description
        );
        if (!teamResult) {
            return Result<TeamOutput>::err(teamResult.error());
        }

        // 3. 永続化
        auto saveResult = repo_->save(teamResult.value());
        if (!saveResult) {
            return Result<TeamOutput>::err(saveResult.error());
        }

        return Result<TeamOutput>::ok(TeamOutput::from(teamResult.value()));
    }

    // ──────────────────────────────────────────────────────────────────────
    // GetTeam — ID でチームを1件取得する（メンバー含む）
    //
    // 対象が存在しない場合はエラー（NotFound）を返す。
    // ──────────────────────────────────────────────────────────────────────
    Result<TeamOutput> getTeam(const GetTeamInput& input) {
        // TeamId を生成（UUID バリデーション）
        auto idResult = TeamId::create(input.teamId);
        if (!idResult) {
            return Result<TeamOutput>::err(idResult.error());
        }

        // Repository から取得
        auto findResult = repo_->findById(idResult.value());
        if (!findResult) {
            return Result<TeamOutput>::err(findResult.error());
        }
        if (!findResult.value().has_value()) {
            return Result<TeamOutput>::err(
                "GetTeam: チームが見つかりません: " + input.teamId
            );
        }

        return Result<TeamOutput>::ok(TeamOutput::from(findResult.value().value()));
    }

    // ──────────────────────────────────────────────────────────────────────
    // ListTeams — 全チーム一覧を取得する（メンバー含む）
    // ──────────────────────────────────────────────────────────────────────
    Result<std::vector<TeamOutput>> listTeams() {
        auto findResult = repo_->findAll();
        if (!findResult) {
            return Result<std::vector<TeamOutput>>::err(findResult.error());
        }

        std::vector<TeamOutput> outputs;
        outputs.reserve(findResult.value().size());
        for (const auto& team : findResult.value()) {
            outputs.push_back(TeamOutput::from(team));
        }

        return Result<std::vector<TeamOutput>>::ok(std::move(outputs));
    }

    // ──────────────────────────────────────────────────────────────────────
    // ListTeamsByUser — 指定ユーザーが所属するチーム一覧を取得する
    // ──────────────────────────────────────────────────────────────────────
    Result<std::vector<TeamOutput>> listTeamsByUser(const ListTeamsByUserInput& input) {
        // UserId を生成（UUID バリデーション）
        auto idResult = UserId::create(input.userId);
        if (!idResult) {
            return Result<std::vector<TeamOutput>>::err(idResult.error());
        }

        auto findResult = repo_->findByUserId(idResult.value());
        if (!findResult) {
            return Result<std::vector<TeamOutput>>::err(findResult.error());
        }

        std::vector<TeamOutput> outputs;
        outputs.reserve(findResult.value().size());
        for (const auto& team : findResult.value()) {
            outputs.push_back(TeamOutput::from(team));
        }

        return Result<std::vector<TeamOutput>>::ok(std::move(outputs));
    }

    // ──────────────────────────────────────────────────────────────────────
    // DeleteTeam — ID でチームを削除する
    //
    // 対象が存在しない場合もエラーにしない（冪等性を保証）。
    // ──────────────────────────────────────────────────────────────────────
    Result<void> deleteTeam(const DeleteTeamInput& input) {
        // TeamId を生成（UUID バリデーション）
        auto idResult = TeamId::create(input.teamId);
        if (!idResult) {
            return Result<void>::err(idResult.error());
        }

        return repo_->remove(idResult.value());
    }

    // ──────────────────────────────────────────────────────────────────────
    // AddMember — チームにメンバーを追加する
    //
    // 処理フロー:
    //   1. チームを取得（存在チェック）
    //   2. TeamMemberId を新規生成
    //   3. TeamMember エンティティを生成
    //   4. Team.addMember()（重複チェック付き）
    //   5. Repository#save で永続化
    // ──────────────────────────────────────────────────────────────────────
    Result<void> addMember(const AddMemberInput& input) {
        // 1. TeamId を生成（UUID バリデーション）
        auto teamIdResult = TeamId::create(input.teamId);
        if (!teamIdResult) {
            return Result<void>::err(teamIdResult.error());
        }

        // 2. UserId を生成（UUID バリデーション）
        auto userIdResult = UserId::create(input.userId);
        if (!userIdResult) {
            return Result<void>::err(userIdResult.error());
        }

        // 3. Role を生成（バリデーション）
        auto roleResult = Role::create(input.role);
        if (!roleResult) {
            return Result<void>::err(roleResult.error());
        }

        // 4. チームを取得（存在チェック）
        auto findResult = repo_->findById(teamIdResult.value());
        if (!findResult) {
            return Result<void>::err(findResult.error());
        }
        if (!findResult.value().has_value()) {
            return Result<void>::err(
                "AddMember: チームが見つかりません: " + input.teamId
            );
        }
        Team team = std::move(findResult.value().value());

        // 5. TeamMemberId を新規生成
        auto memberIdResult = TeamMemberId::generate();
        if (!memberIdResult) {
            return Result<void>::err(memberIdResult.error());
        }

        // 6. TeamMember エンティティを生成
        auto memberResult = TeamMember::create(
            memberIdResult.value(),
            teamIdResult.value(),
            userIdResult.value(),
            roleResult.value()
        );
        if (!memberResult) {
            return Result<void>::err(memberResult.error());
        }

        // 7. Team にメンバーを追加（重複チェック付き）
        auto addResult = team.addMember(std::move(memberResult.value()));
        if (!addResult) {
            return Result<void>::err(addResult.error());
        }

        // 8. 永続化
        return repo_->save(team);
    }

    // ──────────────────────────────────────────────────────────────────────
    // RemoveMember — チームからメンバーを削除する
    //
    // 処理フロー:
    //   1. チームを取得（存在チェック）
    //   2. Team.removeMember()（admin 最低1人ルール含むチェック付き）
    //   3. Repository#save で永続化
    // ──────────────────────────────────────────────────────────────────────
    Result<void> removeMember(const RemoveMemberInput& input) {
        // 1. TeamId を生成（UUID バリデーション）
        auto teamIdResult = TeamId::create(input.teamId);
        if (!teamIdResult) {
            return Result<void>::err(teamIdResult.error());
        }

        // 2. UserId を生成（UUID バリデーション）
        auto userIdResult = UserId::create(input.userId);
        if (!userIdResult) {
            return Result<void>::err(userIdResult.error());
        }

        // 3. チームを取得（存在チェック）
        auto findResult = repo_->findById(teamIdResult.value());
        if (!findResult) {
            return Result<void>::err(findResult.error());
        }
        if (!findResult.value().has_value()) {
            return Result<void>::err(
                "RemoveMember: チームが見つかりません: " + input.teamId
            );
        }
        Team team = std::move(findResult.value().value());

        // 4. メンバーを削除（admin 最低1人ルール含む）
        auto removeResult = team.removeMember(userIdResult.value());
        if (!removeResult) {
            return Result<void>::err(removeResult.error());
        }

        // 5. 永続化
        return repo_->save(team);
    }

    // ──────────────────────────────────────────────────────────────────────
    // ChangeMemberRole — チームメンバーのロールを変更する
    //
    // 処理フロー:
    //   1. チームを取得（存在チェック）
    //   2. Team.changeMemberRole()（admin 最低1人ルール含むチェック付き）
    //   3. Repository#save で永続化
    // ──────────────────────────────────────────────────────────────────────
    Result<void> changeMemberRole(const ChangeMemberRoleInput& input) {
        // 1. TeamId を生成（UUID バリデーション）
        auto teamIdResult = TeamId::create(input.teamId);
        if (!teamIdResult) {
            return Result<void>::err(teamIdResult.error());
        }

        // 2. UserId を生成（UUID バリデーション）
        auto userIdResult = UserId::create(input.userId);
        if (!userIdResult) {
            return Result<void>::err(userIdResult.error());
        }

        // 3. Role を生成（バリデーション）
        auto roleResult = Role::create(input.role);
        if (!roleResult) {
            return Result<void>::err(roleResult.error());
        }

        // 4. チームを取得（存在チェック）
        auto findResult = repo_->findById(teamIdResult.value());
        if (!findResult) {
            return Result<void>::err(findResult.error());
        }
        if (!findResult.value().has_value()) {
            return Result<void>::err(
                "ChangeMemberRole: チームが見つかりません: " + input.teamId
            );
        }
        Team team = std::move(findResult.value().value());

        // 5. ロール変更（admin 最低1人ルール含む）
        auto changeResult = team.changeMemberRole(userIdResult.value(), roleResult.value());
        if (!changeResult) {
            return Result<void>::err(changeResult.error());
        }

        // 6. 永続化
        return repo_->save(team);
    }

private:
    std::shared_ptr<ITeamRepository> repo_;
};
