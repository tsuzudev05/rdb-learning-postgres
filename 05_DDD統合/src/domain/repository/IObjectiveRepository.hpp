#pragma once

#include <optional>
#include <vector>
#include "../model/objective/Objective.hpp"
#include "../model/objective/ObjectiveId.hpp"
#include "../model/period/PeriodId.hpp"
#include "../model/user/UserId.hpp"
#include "../../common/Result.hpp"

// =============================================================================
// IObjectiveRepository — 目標Repositoryインターフェース（純粋仮想）
//
// ドメイン層に属する。インフラ層への依存を持たない。
// 実装は infrastructure/repository/PgObjectiveRepository で行う。
//
// 操作一覧:
//   - findById       : ID で目標を1件取得
//   - findByPeriodId : 期間IDに紐づく目標一覧を取得（display_order 昇順）
//   - findByOwnerId  : オーナーIDに紐づく目標一覧を取得
//   - save           : 新規作成 or 更新
//   - remove         : ID で目標を削除
// =============================================================================
class IObjectiveRepository {
public:
    virtual ~IObjectiveRepository() = default;

    // ID で目標を取得。存在しない場合は std::nullopt を返す。
    virtual Result<std::optional<Objective>> findById(const ObjectiveId& id) const = 0;

    // 期間IDに紐づく目標一覧を取得（display_order 昇順）。
    virtual Result<std::vector<Objective>> findByPeriodId(const PeriodId& periodId) const = 0;

    // オーナーIDに紐づく目標一覧を取得。
    virtual Result<std::vector<Objective>> findByOwnerId(const UserId& ownerId) const = 0;

    // 目標を保存（新規作成 or 更新）。
    virtual Result<void> save(const Objective& objective) = 0;

    // ID で目標を削除。対象が存在しない場合もエラーにしない。
    virtual Result<void> remove(const ObjectiveId& id) = 0;
};
