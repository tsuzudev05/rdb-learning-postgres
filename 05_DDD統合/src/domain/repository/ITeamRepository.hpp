#pragma once

#include <optional>
#include <vector>
#include "../model/team/Team.hpp"
#include "../model/team/TeamId.hpp"
#include "../model/user/UserId.hpp"
#include "../../common/Result.hpp"

// =============================================================================
// ITeamRepository — チームRepositoryインターフェース（純粋仮想）
//
// ドメイン層に属する。インフラ層への依存を持たない。
// 実装は infrastructure/repository/PgTeamRepository で行う。
//
// Team 集約は Team + TeamMember を一括で管理する（集約整合性の保証）。
// TeamMember は Team 集約内でのみ管理し、個別のRepositoryは持たない。
//
// 操作一覧:
//   - findById      : ID でチームを1件取得（TeamMemberも含む）
//   - findByUserId  : 指定ユーザーが所属するチーム一覧を取得
//   - findAll       : 全チームを取得
//   - save          : 新規作成 or 更新（Team + TeamMember を一括保存）
//   - remove        : ID でチームを削除
// =============================================================================
class ITeamRepository {
public:
    virtual ~ITeamRepository() = default;

    // ID でチームを取得（TeamMember 含む）。存在しない場合は std::nullopt を返す。
    virtual Result<std::optional<Team>> findById(const TeamId& id) const = 0;

    // 指定ユーザーが所属するチーム一覧を取得（TeamMember 含む）。
    virtual Result<std::vector<Team>> findByUserId(const UserId& userId) const = 0;

    // 全チームを取得（TeamMember 含む）。
    virtual Result<std::vector<Team>> findAll() const = 0;

    // チームを保存（新規作成 or 更新）。Team + TeamMember を一括保存する。
    virtual Result<void> save(const Team& team) = 0;

    // ID でチームを削除。対象が存在しない場合もエラーにしない。
    virtual Result<void> remove(const TeamId& id) = 0;
};
