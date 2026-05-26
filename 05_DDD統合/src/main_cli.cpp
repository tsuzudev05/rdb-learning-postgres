// =============================================================================
// main_cli.cpp — CLI エントリーポイント
//
// OKR管理ツールの対話型 CLI を起動する。
// スモークテスト（main.cpp）とは分離し、実用的な操作を可能にする。
//
// DevContainer 内で以下のコマンドで実行:
//   cd /workspace/05_DDD統合
//   make cli
//
// または直接（DevContainer内）:
//   cd /workspace/05_DDD統合
//   make cli
//   DATABASE_URL="postgresql://postgres:pass@postgres:5432/learning" ./build/okr-cli
//
// 環境変数:
//   DATABASE_URL  接続文字列（未設定時は "postgresql://postgres:pass@postgres:5432/learning"）
// =============================================================================

#include <iostream>
#include <memory>
#include <cstdlib>

#include <pqxx/pqxx>

// Infrastructure（libpqxx 実装）
#include "infrastructure/repository/PgUserRepository.hpp"
#include "infrastructure/repository/PgTeamRepository.hpp"
#include "infrastructure/repository/PgPeriodRepository.hpp"
#include "infrastructure/repository/PgObjectiveRepository.hpp"

// Application（UseCase）
#include "application/usecase/UserUseCase.hpp"
#include "application/usecase/TeamUseCase.hpp"
#include "application/usecase/PeriodUseCase.hpp"
#include "application/usecase/ObjectiveUseCase.hpp"

// Presentation（CLI）
#include "presentation/cli/CliApp.hpp"

// ─── 接続文字列 ──────────────────────────────────────────────────────────────
static std::string connectionString() {
    const char* url = std::getenv("DATABASE_URL");
    if (url && url[0] != '\0') return url;
    return "postgresql://postgres:pass@postgres:5432/learning";
}

// ─── main ────────────────────────────────────────────────────────────────────
int main() {
    try {
        // ── 1. DB 接続を確立 ──────────────────────────────────────────────────
        auto conn = std::make_shared<pqxx::connection>(connectionString());

        // ── 2. Repository を生成（インフラ層）────────────────────────────────
        auto userRepo      = std::make_shared<PgUserRepository>(conn);
        auto teamRepo      = std::make_shared<PgTeamRepository>(conn);
        auto periodRepo    = std::make_shared<PgPeriodRepository>(conn);
        auto objectiveRepo = std::make_shared<PgObjectiveRepository>(conn);

        // ── 3. UseCase を生成（アプリケーション層）───────────────────────────
        auto userUC      = std::make_shared<UserUseCase>(userRepo);
        auto teamUC      = std::make_shared<TeamUseCase>(teamRepo);
        auto periodUC    = std::make_shared<PeriodUseCase>(periodRepo);
        auto objectiveUC = std::make_shared<ObjectiveUseCase>(objectiveRepo);

        // ── 4. CLI を起動（プレゼンテーション層）─────────────────────────────
        CliApp app{userUC, teamUC, periodUC, objectiveUC};
        app.run();

        return 0;

    } catch (const std::exception& e) {
        std::cerr << "❌ 起動エラー: " << e.what() << "\n";
        std::cerr << "   DATABASE_URL が正しく設定されているか確認してください。\n";
        return 1;
    }
}
