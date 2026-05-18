#pragma once

#include <optional>
#include <vector>
#include "../model/keyresult/KeyResult.hpp"
#include "../model/keyresult/KeyResultId.hpp"
#include "../model/objective/ObjectiveId.hpp"
#include "../model/user/UserId.hpp"
#include "../../common/Result.hpp"

// =============================================================================
// IKeyResultRepository — 主要な結果Repositoryインターフェース（純粋仮想）
//
// ドメイン層に属する。インフラ層への依存を持たない。
// 実装は infrastructure/repository/PgKeyResultRepository で行う。
//
// KeyResult 集約は KeyResult + KrProgressLog を一括で管理する。
// KrProgressLog は KeyResult 集約内でのみ管理し、個別のRepositoryは持たない。
//
// 操作一覧:
//   - findById          : ID でKRを1件取得（KrProgressLog含む）
//   - findByObjectiveId : 目標IDに紐づくKR一覧を取得（display_order 昇順）
//   - findByOwnerId     : オーナーIDに紐づくKR一覧を取得
//   - save              : 新規作成 or 更新（KeyResult + KrProgressLog を一括保存）
//   - remove            : ID でKRを削除
// =============================================================================
class IKeyResultRepository {
public:
    virtual ~IKeyResultRepository() = default;

    // ID でKRを取得（KrProgressLog 含む）。存在しない場合は std::nullopt を返す。
    virtual Result<std::optional<KeyResult>> findById(const KeyResultId& id) const = 0;

    // 目標IDに紐づくKR一覧を取得（KrProgressLog 含む、display_order 昇順）。
    virtual Result<std::vector<KeyResult>> findByObjectiveId(const ObjectiveId& objectiveId) const = 0;

    // オーナーIDに紐づくKR一覧を取得（KrProgressLog 含む）。
    virtual Result<std::vector<KeyResult>> findByOwnerId(const UserId& ownerId) const = 0;

    // KRを保存（新規作成 or 更新）。KeyResult + KrProgressLog を一括保存する。
    virtual Result<void> save(const KeyResult& keyResult) = 0;

    // ID でKRを削除。対象が存在しない場合もエラーにしない。
    virtual Result<void> remove(const KeyResultId& id) = 0;
};
