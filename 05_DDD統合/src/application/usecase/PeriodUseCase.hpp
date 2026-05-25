#pragma once

#include <memory>
#include <string>
#include <optional>
#include <vector>
#include "../../domain/model/period/Period.hpp"
#include "../../domain/model/period/PeriodId.hpp"
#include "../../domain/model/period/Half.hpp"
#include "../../domain/model/period/DateRange.hpp"
#include "../../domain/model/team/TeamId.hpp"
#include "../../domain/repository/IPeriodRepository.hpp"
#include "../../common/Result.hpp"

// =============================================================================
// PeriodUseCase — 期間ユースケース（アプリケーション層）
//
// ドメイン層の IPeriodRepository に依存注入（DI）パターンで依存する。
// インフラ層（libpqxx）への直接依存を持たない。
//
// 提供するユースケース:
//   - CreatePeriod        : 新規期間（半期OKRサイクル）を作成する
//   - GetPeriod           : ID で期間を1件取得する
//   - ListPeriods         : 全期間一覧を取得する
//   - ListPeriodsByTeam   : 指定チームに紐づく期間一覧を取得する
//   - DeletePeriod        : ID で期間を削除する
//
// エラーハンドリング:
//   すべての操作は Result<T> を返す。例外は使わない。
// =============================================================================

// ─── 入力 DTO ──────────────────────────────────────────────────────────────

struct CreatePeriodInput {
    std::string teamId;     // UUID 文字列
    std::string name;       // 例: "2025-H1"
    std::string half;       // "H1" or "H2"
    std::string startDate;  // "YYYY-MM-DD" 形式
    std::string endDate;    // "YYYY-MM-DD" 形式
};

struct GetPeriodInput {
    std::string periodId;   // UUID 文字列
};

struct DeletePeriodInput {
    std::string periodId;   // UUID 文字列
};

struct ListPeriodsByTeamInput {
    std::string teamId;     // UUID 文字列
};

// ─── 出力 DTO ──────────────────────────────────────────────────────────────

struct PeriodOutput {
    std::string id;
    std::string teamId;
    std::string name;
    std::string half;       // "H1" or "H2"
    std::string startDate;  // "YYYY-MM-DD"
    std::string endDate;    // "YYYY-MM-DD"
    std::string createdAt;
    std::string updatedAt;

    // Period エンティティから変換するファクトリ
    static PeriodOutput from(const Period& period) {
        return PeriodOutput{
            period.id().value(),
            period.teamId().value(),
            period.name(),
            period.half().toString(),
            period.dateRange().startDate(),
            period.dateRange().endDate(),
            period.createdAt(),
            period.updatedAt()
        };
    }
};

// ─── PeriodUseCase ──────────────────────────────────────────────────────────

class PeriodUseCase {
public:
    // コンストラクタ：IPeriodRepository を DI で受け取る（所有権は共有しない）
    explicit PeriodUseCase(std::shared_ptr<IPeriodRepository> repo)
        : repo_(std::move(repo)) {}

    // ──────────────────────────────────────────────────────────────────────
    // CreatePeriod — 新規期間を作成する
    //
    // 処理フロー:
    //   1. 入力値を各値オブジェクトに変換（バリデーション含む）
    //   2. PeriodId を新規生成
    //   3. Period エンティティを生成
    //   4. Repository#save で永続化
    // ──────────────────────────────────────────────────────────────────────
    Result<PeriodOutput> createPeriod(const CreatePeriodInput& input) {
        // 1. TeamId を生成（UUID バリデーション）
        auto teamIdResult = TeamId::create(input.teamId);
        if (!teamIdResult) {
            return Result<PeriodOutput>::err(teamIdResult.error());
        }

        // 2. Half を生成（H1 / H2 バリデーション）
        auto halfResult = Half::create(input.half);
        if (!halfResult) {
            return Result<PeriodOutput>::err(halfResult.error());
        }

        // 3. DateRange を生成（日付フォーマット・start < end バリデーション）
        auto dateRangeResult = DateRange::create(input.startDate, input.endDate);
        if (!dateRangeResult) {
            return Result<PeriodOutput>::err(dateRangeResult.error());
        }

        // 4. PeriodId を新規生成
        auto idResult = PeriodId::generate();
        if (!idResult) {
            return Result<PeriodOutput>::err(idResult.error());
        }

        // 5. Period エンティティを生成（name バリデーション含む）
        auto periodResult = Period::create(
            idResult.value(),
            teamIdResult.value(),
            input.name,
            halfResult.value(),
            dateRangeResult.value()
        );
        if (!periodResult) {
            return Result<PeriodOutput>::err(periodResult.error());
        }

        // 6. 永続化
        auto saveResult = repo_->save(periodResult.value());
        if (!saveResult) {
            return Result<PeriodOutput>::err(saveResult.error());
        }

        return Result<PeriodOutput>::ok(PeriodOutput::from(periodResult.value()));
    }

    // ──────────────────────────────────────────────────────────────────────
    // GetPeriod — ID で期間を1件取得する
    //
    // 対象が存在しない場合はエラー（NotFound）を返す。
    // ──────────────────────────────────────────────────────────────────────
    Result<PeriodOutput> getPeriod(const GetPeriodInput& input) {
        // PeriodId を生成（UUID バリデーション）
        auto idResult = PeriodId::create(input.periodId);
        if (!idResult) {
            return Result<PeriodOutput>::err(idResult.error());
        }

        // Repository から取得
        auto findResult = repo_->findById(idResult.value());
        if (!findResult) {
            return Result<PeriodOutput>::err(findResult.error());
        }
        if (!findResult.value().has_value()) {
            return Result<PeriodOutput>::err(
                "GetPeriod: 期間が見つかりません: " + input.periodId
            );
        }

        return Result<PeriodOutput>::ok(PeriodOutput::from(findResult.value().value()));
    }

    // ──────────────────────────────────────────────────────────────────────
    // ListPeriods — 全期間一覧を取得する
    // ──────────────────────────────────────────────────────────────────────
    Result<std::vector<PeriodOutput>> listPeriods() {
        auto findResult = repo_->findAll();
        if (!findResult) {
            return Result<std::vector<PeriodOutput>>::err(findResult.error());
        }

        std::vector<PeriodOutput> outputs;
        outputs.reserve(findResult.value().size());
        for (const auto& period : findResult.value()) {
            outputs.push_back(PeriodOutput::from(period));
        }

        return Result<std::vector<PeriodOutput>>::ok(std::move(outputs));
    }

    // ──────────────────────────────────────────────────────────────────────
    // ListPeriodsByTeam — 指定チームに紐づく期間一覧を取得する
    // ──────────────────────────────────────────────────────────────────────
    Result<std::vector<PeriodOutput>> listPeriodsByTeam(const ListPeriodsByTeamInput& input) {
        // TeamId を生成（UUID バリデーション）
        auto idResult = TeamId::create(input.teamId);
        if (!idResult) {
            return Result<std::vector<PeriodOutput>>::err(idResult.error());
        }

        auto findResult = repo_->findByTeamId(idResult.value());
        if (!findResult) {
            return Result<std::vector<PeriodOutput>>::err(findResult.error());
        }

        std::vector<PeriodOutput> outputs;
        outputs.reserve(findResult.value().size());
        for (const auto& period : findResult.value()) {
            outputs.push_back(PeriodOutput::from(period));
        }

        return Result<std::vector<PeriodOutput>>::ok(std::move(outputs));
    }

    // ──────────────────────────────────────────────────────────────────────
    // DeletePeriod — ID で期間を削除する
    //
    // 対象が存在しない場合もエラーにしない（冪等性を保証）。
    // ──────────────────────────────────────────────────────────────────────
    Result<void> deletePeriod(const DeletePeriodInput& input) {
        // PeriodId を生成（UUID バリデーション）
        auto idResult = PeriodId::create(input.periodId);
        if (!idResult) {
            return Result<void>::err(idResult.error());
        }

        return repo_->remove(idResult.value());
    }

private:
    std::shared_ptr<IPeriodRepository> repo_;
};
