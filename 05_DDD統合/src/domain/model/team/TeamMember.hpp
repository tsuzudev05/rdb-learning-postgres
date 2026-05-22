#pragma once

#include <string>
#include <utility>
#include "TeamMemberId.hpp"
#include "TeamId.hpp"
#include "../user/UserId.hpp"
#include "../team/Role.hpp"
#include "../../../common/Result.hpp"

// =============================================================================
// TeamMember — チームメンバーエンティティ（Team 集約内）
//
// ドメインルール:
//   - team_members テーブルの 1 行に対応
//   - ロール変更はドメイン操作として提供する
//   - 同一 team に同一 user が重複して存在してはならない
//     → ただし重複排除のチェックは Team 集約ルート（Team クラス）で行う
// =============================================================================
class TeamMember {
public:
    // ファクトリ：新規メンバー生成
    static Result<TeamMember> create(
        TeamMemberId id,
        TeamId       teamId,
        UserId       userId,
        Role         role,
        std::string  joinedAt = ""
    ) {
        return Result<TeamMember>::ok(TeamMember{
            std::move(id),
            std::move(teamId),
            std::move(userId),
            std::move(role),
            std::move(joinedAt)
        });
    }

    // ─── アクセサ ───────────────────────────────────────────
    const TeamMemberId& id()       const { return id_; }
    const TeamId&       teamId()   const { return teamId_; }
    const UserId&       userId()   const { return userId_; }
    const Role&         role()     const { return role_; }
    const std::string&  joinedAt() const { return joinedAt_; }

    // ─── ドメイン操作 ────────────────────────────────────────
    // ロール変更
    void changeRole(Role newRole) {
        role_ = std::move(newRole);
    }

    // ─── 等価比較（ID で同一性を判定）───────────────────────
    bool operator==(const TeamMember& other) const { return id_ == other.id_; }
    bool operator!=(const TeamMember& other) const { return !(*this == other); }

private:
    TeamMember(TeamMemberId id, TeamId teamId, UserId userId, Role role, std::string joinedAt)
        : id_(std::move(id))
        , teamId_(std::move(teamId))
        , userId_(std::move(userId))
        , role_(std::move(role))
        , joinedAt_(std::move(joinedAt))
    {}

    TeamMemberId id_;
    TeamId       teamId_;
    UserId       userId_;
    Role         role_;
    std::string  joinedAt_;
};
