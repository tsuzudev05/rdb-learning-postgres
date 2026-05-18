#pragma once

#include <optional>
#include <vector>
#include "../model/period/Period.hpp"
#include "../model/period/PeriodId.hpp"
#include "../model/team/TeamId.hpp"
#include "../../common/Result.hpp"

// =============================================================================
// IPeriodRepository — 期間Repositoryインターフェース（純粋仮想）
//
// ドメイン層に属する。インフラ層への依存を持たない。
// 実装は infrastructure/repository/PgPeriodRepository で行う。
//
// 操作一覧:
//   - findById     : ID で期間を1件取得
//   - findByTeamId : チームIDに紐づく期間一覧を取得
//   - findAll      : 全期間を取得
//   - save         : 新規作成 or 更新
//   - remove       : ID で期間を削除
// =============================================================================
class IPeriodRepository {
public:
    virtual ~IPeriodRepository() = default;

    // ID で期間を取得。存在しない場合は std::nullopt を返す。
    virtual Result<std::optional<Period>> findById(const PeriodId& id) const = 0;

    // チームIDに紐づく期間一覧を取得。
    virtual Result<std::vector<Period>> findByTeamId(const TeamId& teamId) const = 0;

    // 全期間を取得。
    virtual Result<std::vector<Period>> findAll() const = 0;

    // 期間を保存（新規作成 or 更新）。
    virtual Result<void> save(const Period& period) = 0;

    // ID で期間を削除。対象が存在しない場合もエラーにしない。
    virtual Result<void> remove(const PeriodId& id) = 0;
};
