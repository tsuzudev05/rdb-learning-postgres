#pragma once

// =============================================================================
// CliApp.hpp — CLI プレゼンテーション層
//
// OKR管理ツールの対話型 CLI インターフェース。
// 標準入力からコマンドを受け取り、UseCase を呼び出して結果を表示する。
//
// 対応コマンド:
//   user create <name> <email>
//   user list
//   user get <id>
//   user delete <id>
//   team create <name>
//   team list
//   team get <id>
//   team add-member <team-id> <user-id> <role>
//   period create <team-id> <name> <half> <start-date> <end-date>
//   period list
//   period list-by-team <team-id>
//   objective create <period-id> <owner-id> <title>
//   objective list <period-id>
//   objective get <id>
//   help
//   quit / exit
//
// 設計方針:
//   - UseCase を shared_ptr で DI（インフラ層に依存しない）
//   - すべての出力は std::ostream& に書き込む（テスト可能）
//   - Result<T> の失敗はエラーとして out に出力し、ループを継続する
// =============================================================================

#include <iostream>
#include <sstream>
#include <string>
#include <vector>
#include <memory>
#include <iomanip>

#include "../../application/usecase/UserUseCase.hpp"
#include "../../application/usecase/TeamUseCase.hpp"
#include "../../application/usecase/PeriodUseCase.hpp"
#include "../../application/usecase/ObjectiveUseCase.hpp"

// ─── CliApp ──────────────────────────────────────────────────────────────────

class CliApp {
public:
    // ──────────────────────────────────────────────────────────────────────────
    // コンストラクタ: UseCase を DI で受け取る
    // ──────────────────────────────────────────────────────────────────────────
    CliApp(
        std::shared_ptr<UserUseCase>      userUC,
        std::shared_ptr<TeamUseCase>      teamUC,
        std::shared_ptr<PeriodUseCase>    periodUC,
        std::shared_ptr<ObjectiveUseCase> objectiveUC,
        std::ostream&                     out = std::cout
    )
        : userUC_(std::move(userUC))
        , teamUC_(std::move(teamUC))
        , periodUC_(std::move(periodUC))
        , objectiveUC_(std::move(objectiveUC))
        , out_(out)
    {}

    // ──────────────────────────────────────────────────────────────────────────
    // run() — メインループ
    //
    // プロンプトを表示し、1行ずつコマンドを読み込んでディスパッチする。
    // "quit" または "exit" で終了する。EOF でも終了する。
    // ──────────────────────────────────────────────────────────────────────────
    void run(std::istream& in = std::cin) {
        printBanner();
        std::string line;
        while (true) {
            out_ << "\nokr> ";
            out_.flush();
            if (!std::getline(in, line)) break;

            // 空行・空白のみは無視
            auto tokens = tokenize(line);
            if (tokens.empty()) continue;

            const std::string& cmd = tokens[0];
            if (cmd == "quit" || cmd == "exit") {
                out_ << "bye.\n";
                break;
            }
            dispatch(tokens);
        }
    }

private:
    std::shared_ptr<UserUseCase>      userUC_;
    std::shared_ptr<TeamUseCase>      teamUC_;
    std::shared_ptr<PeriodUseCase>    periodUC_;
    std::shared_ptr<ObjectiveUseCase> objectiveUC_;
    std::ostream&                     out_;

    // ──────────────────────────────────────────────────────────────────────────
    // コマンドディスパッチ
    // ──────────────────────────────────────────────────────────────────────────
    void dispatch(const std::vector<std::string>& tokens) {
        const std::string& ns = tokens[0];

        if (ns == "help") {
            printHelp();
        } else if (ns == "user") {
            dispatchUser(tokens);
        } else if (ns == "team") {
            dispatchTeam(tokens);
        } else if (ns == "period") {
            dispatchPeriod(tokens);
        } else if (ns == "objective") {
            dispatchObjective(tokens);
        } else {
            out_ << "❌ 不明なコマンド: " << ns << "  (help で使い方を確認)\n";
        }
    }

    // ──────────────────────────────────────────────────────────────────────────
    // user サブコマンド
    // ──────────────────────────────────────────────────────────────────────────
    void dispatchUser(const std::vector<std::string>& tokens) {
        if (tokens.size() < 2) { out_ << "使い方: user <create|list|get|delete>\n"; return; }
        const std::string& sub = tokens[1];

        if (sub == "create") {
            // user create <name> <email>
            if (tokens.size() < 4) {
                out_ << "使い方: user create <name> <email>\n"; return;
            }
            CreateUserInput input;
            input.name         = tokens[2];
            input.email        = tokens[3];
            input.passwordHash = "cli_placeholder_hash";  // CLI では仮パスワードハッシュ

            auto result = userUC_->createUser(input);
            if (!result) {
                out_ << "❌ " << result.error() << "\n"; return;
            }
            const auto& u = result.value();
            out_ << "✅ ユーザーを作成しました\n";
            printUser(u);

        } else if (sub == "list") {
            // user list
            auto result = userUC_->listUsers();
            if (!result) {
                out_ << "❌ " << result.error() << "\n"; return;
            }
            const auto& users = result.value();
            if (users.empty()) { out_ << "（ユーザーなし）\n"; return; }
            out_ << "ユーザー一覧（" << users.size() << "件）\n";
            printSeparator();
            for (const auto& u : users) printUser(u);

        } else if (sub == "get") {
            // user get <id>
            if (tokens.size() < 3) { out_ << "使い方: user get <id>\n"; return; }
            auto result = userUC_->getUser(GetUserInput{tokens[2]});
            if (!result) {
                out_ << "❌ " << result.error() << "\n"; return;
            }
            printUser(result.value());

        } else if (sub == "delete") {
            // user delete <id>
            if (tokens.size() < 3) { out_ << "使い方: user delete <id>\n"; return; }
            auto result = userUC_->deleteUser(DeleteUserInput{tokens[2]});
            if (!result) {
                out_ << "❌ " << result.error() << "\n"; return;
            }
            out_ << "✅ ユーザーを削除しました: " << tokens[2] << "\n";

        } else {
            out_ << "❌ 不明なサブコマンド: user " << sub << "\n";
        }
    }

    // ──────────────────────────────────────────────────────────────────────────
    // team サブコマンド
    // ──────────────────────────────────────────────────────────────────────────
    void dispatchTeam(const std::vector<std::string>& tokens) {
        if (tokens.size() < 2) { out_ << "使い方: team <create|list|get|add-member>\n"; return; }
        const std::string& sub = tokens[1];

        if (sub == "create") {
            // team create <name> [description]
            if (tokens.size() < 3) { out_ << "使い方: team create <name> [description]\n"; return; }
            CreateTeamInput input;
            input.name = tokens[2];
            if (tokens.size() >= 4) input.description = tokens[3];

            auto result = teamUC_->createTeam(input);
            if (!result) { out_ << "❌ " << result.error() << "\n"; return; }
            out_ << "✅ チームを作成しました\n";
            printTeam(result.value());

        } else if (sub == "list") {
            // team list
            auto result = teamUC_->listTeams();
            if (!result) { out_ << "❌ " << result.error() << "\n"; return; }
            const auto& teams = result.value();
            if (teams.empty()) { out_ << "（チームなし）\n"; return; }
            out_ << "チーム一覧（" << teams.size() << "件）\n";
            printSeparator();
            for (const auto& t : teams) printTeam(t);

        } else if (sub == "get") {
            // team get <id>
            if (tokens.size() < 3) { out_ << "使い方: team get <id>\n"; return; }
            auto result = teamUC_->getTeam(GetTeamInput{tokens[2]});
            if (!result) { out_ << "❌ " << result.error() << "\n"; return; }
            printTeam(result.value());

        } else if (sub == "add-member") {
            // team add-member <team-id> <user-id> <role>
            if (tokens.size() < 5) {
                out_ << "使い方: team add-member <team-id> <user-id> <role(admin|member)>\n"; return;
            }
            AddMemberInput input{tokens[2], tokens[3], tokens[4]};
            auto result = teamUC_->addMember(input);
            if (!result) { out_ << "❌ " << result.error() << "\n"; return; }
            out_ << "✅ メンバーを追加しました\n";
            // 追加後の状態を表示するため team get で再取得
            auto getResult = teamUC_->getTeam(GetTeamInput{tokens[2]});
            if (getResult) printTeam(getResult.value());

        } else {
            out_ << "❌ 不明なサブコマンド: team " << sub << "\n";
        }
    }

    // ──────────────────────────────────────────────────────────────────────────
    // period サブコマンド
    // ──────────────────────────────────────────────────────────────────────────
    void dispatchPeriod(const std::vector<std::string>& tokens) {
        if (tokens.size() < 2) {
            out_ << "使い方: period <create|list|list-by-team|get|delete>\n"; return;
        }
        const std::string& sub = tokens[1];

        if (sub == "create") {
            // period create <team-id> <name> <half> <start-date> <end-date>
            // 例: period create <uuid> 2025-H1 H1 2025-01-01 2025-06-30
            if (tokens.size() < 7) {
                out_ << "使い方: period create <team-id> <name> <half(H1|H2)> <start-date(YYYY-MM-DD)> <end-date(YYYY-MM-DD)>\n";
                return;
            }
            CreatePeriodInput input;
            input.teamId    = tokens[2];
            input.name      = tokens[3];
            input.half      = tokens[4];
            input.startDate = tokens[5];
            input.endDate   = tokens[6];

            auto result = periodUC_->createPeriod(input);
            if (!result) { out_ << "❌ " << result.error() << "\n"; return; }
            out_ << "✅ 期間を作成しました\n";
            printPeriod(result.value());

        } else if (sub == "list") {
            // period list
            auto result = periodUC_->listPeriods();
            if (!result) { out_ << "❌ " << result.error() << "\n"; return; }
            const auto& periods = result.value();
            if (periods.empty()) { out_ << "（期間なし）\n"; return; }
            out_ << "期間一覧（" << periods.size() << "件）\n";
            printSeparator();
            for (const auto& p : periods) printPeriod(p);

        } else if (sub == "list-by-team") {
            // period list-by-team <team-id>
            if (tokens.size() < 3) { out_ << "使い方: period list-by-team <team-id>\n"; return; }
            auto result = periodUC_->listPeriodsByTeam(ListPeriodsByTeamInput{tokens[2]});
            if (!result) { out_ << "❌ " << result.error() << "\n"; return; }
            const auto& periods = result.value();
            if (periods.empty()) { out_ << "（期間なし）\n"; return; }
            out_ << "期間一覧（チーム=" << tokens[2] << ", " << periods.size() << "件）\n";
            printSeparator();
            for (const auto& p : periods) printPeriod(p);

        } else if (sub == "get") {
            // period get <id>
            if (tokens.size() < 3) { out_ << "使い方: period get <id>\n"; return; }
            auto result = periodUC_->getPeriod(GetPeriodInput{tokens[2]});
            if (!result) { out_ << "❌ " << result.error() << "\n"; return; }
            printPeriod(result.value());

        } else if (sub == "delete") {
            // period delete <id>
            if (tokens.size() < 3) { out_ << "使い方: period delete <id>\n"; return; }
            auto result = periodUC_->deletePeriod(DeletePeriodInput{tokens[2]});
            if (!result) { out_ << "❌ " << result.error() << "\n"; return; }
            out_ << "✅ 期間を削除しました: " << tokens[2] << "\n";

        } else {
            out_ << "❌ 不明なサブコマンド: period " << sub << "\n";
        }
    }

    // ──────────────────────────────────────────────────────────────────────────
    // objective サブコマンド
    // ──────────────────────────────────────────────────────────────────────────
    void dispatchObjective(const std::vector<std::string>& tokens) {
        if (tokens.size() < 2) {
            out_ << "使い方: objective <create|list|get|delete>\n"; return;
        }
        const std::string& sub = tokens[1];

        if (sub == "create") {
            // objective create <period-id> <owner-id> <title> [display-order]
            if (tokens.size() < 5) {
                out_ << "使い方: objective create <period-id> <owner-id> <title> [display-order]\n";
                return;
            }
            CreateObjectiveInput input;
            input.periodId     = tokens[2];
            input.ownerId      = tokens[3];
            input.title        = tokens[4];
            input.displayOrder = (tokens.size() >= 6) ? std::stoi(tokens[5]) : 0;

            auto result = objectiveUC_->createObjective(input);
            if (!result) { out_ << "❌ " << result.error() << "\n"; return; }
            out_ << "✅ 目標を作成しました\n";
            printObjective(result.value());

        } else if (sub == "list") {
            // objective list <period-id>
            if (tokens.size() < 3) { out_ << "使い方: objective list <period-id>\n"; return; }
            auto result = objectiveUC_->listObjectivesByPeriod(
                ListObjectivesByPeriodInput{tokens[2]}
            );
            if (!result) { out_ << "❌ " << result.error() << "\n"; return; }
            const auto& objs = result.value();
            if (objs.empty()) { out_ << "（目標なし）\n"; return; }
            out_ << "目標一覧（期間=" << tokens[2] << ", " << objs.size() << "件）\n";
            printSeparator();
            for (const auto& o : objs) printObjective(o);

        } else if (sub == "get") {
            // objective get <id>
            if (tokens.size() < 3) { out_ << "使い方: objective get <id>\n"; return; }
            auto result = objectiveUC_->getObjective(GetObjectiveInput{tokens[2]});
            if (!result) { out_ << "❌ " << result.error() << "\n"; return; }
            printObjective(result.value());

        } else if (sub == "delete") {
            // objective delete <id>
            if (tokens.size() < 3) { out_ << "使い方: objective delete <id>\n"; return; }
            auto result = objectiveUC_->deleteObjective(DeleteObjectiveInput{tokens[2]});
            if (!result) { out_ << "❌ " << result.error() << "\n"; return; }
            out_ << "✅ 目標を削除しました: " << tokens[2] << "\n";

        } else {
            out_ << "❌ 不明なサブコマンド: objective " << sub << "\n";
        }
    }

    // ──────────────────────────────────────────────────────────────────────────
    // 表示ヘルパー
    // ──────────────────────────────────────────────────────────────────────────

    void printBanner() const {
        out_ << "========================================\n"
             << " OKR管理ツール CLI（フェーズ7）\n"
             << " 'help' でコマンド一覧  'quit' で終了\n"
             << "========================================\n";
    }

    void printSeparator() const {
        out_ << "----------------------------------------\n";
    }

    void printUser(const UserOutput& u) const {
        out_ << "  id        : " << u.id        << "\n"
             << "  name      : " << u.name      << "\n"
             << "  email     : " << u.email     << "\n"
             << "  createdAt : " << u.createdAt << "\n";
    }

    void printTeam(const TeamOutput& t) const {
        out_ << "  id          : " << t.id          << "\n"
             << "  name        : " << t.name        << "\n";
        if (t.description.has_value()) {
            out_ << "  description : " << t.description.value() << "\n";
        }
        out_ << "  members     : " << t.members.size() << "人\n";
        for (const auto& m : t.members) {
            out_ << "    - userId=" << m.userId
                 << "  role=" << m.role << "\n";
        }
    }

    void printPeriod(const PeriodOutput& p) const {
        out_ << "  id        : " << p.id        << "\n"
             << "  teamId    : " << p.teamId    << "\n"
             << "  name      : " << p.name      << "\n"
             << "  half      : " << p.half      << "\n"
             << "  startDate : " << p.startDate << "\n"
             << "  endDate   : " << p.endDate   << "\n";
    }

    void printObjective(const ObjectiveOutput& o) const {
        out_ << "  id           : " << o.id           << "\n"
             << "  title        : " << o.title        << "\n"
             << "  periodId     : " << o.periodId     << "\n"
             << "  ownerId      : " << o.ownerId      << "\n"
             << "  displayOrder : " << o.displayOrder << "\n";
        if (o.description.has_value()) {
            out_ << "  description  : " << o.description.value() << "\n";
        }
    }

    void printHelp() const {
        out_ <<
            "【使用可能なコマンド】\n"
            "\n"
            "  user create <name> <email>                           ユーザーを作成\n"
            "  user list                                            ユーザー一覧\n"
            "  user get <id>                                        ユーザー詳細\n"
            "  user delete <id>                                     ユーザー削除\n"
            "\n"
            "  team create <name> [description]                     チームを作成\n"
            "  team list                                            チーム一覧\n"
            "  team get <id>                                        チーム詳細（メンバー含む）\n"
            "  team add-member <team-id> <user-id> <role>           メンバーを追加\n"
            "\n"
            "  period create <team-id> <name> <H1|H2> <start> <end>  期間を作成\n"
            "  period list                                          期間一覧（全件）\n"
            "  period list-by-team <team-id>                        期間一覧（チーム別）\n"
            "  period get <id>                                      期間詳細\n"
            "  period delete <id>                                   期間削除\n"
            "\n"
            "  objective create <period-id> <owner-id> <title> [order]  目標を作成\n"
            "  objective list <period-id>                           目標一覧（期間別）\n"
            "  objective get <id>                                   目標詳細\n"
            "  objective delete <id>                                目標削除\n"
            "\n"
            "  help                                                 このヘルプを表示\n"
            "  quit / exit                                          終了\n";
    }

    // ──────────────────────────────────────────────────────────────────────────
    // tokenize — 空白区切りでトークン分割
    // ──────────────────────────────────────────────────────────────────────────
    static std::vector<std::string> tokenize(const std::string& line) {
        std::vector<std::string> tokens;
        std::istringstream iss(line);
        std::string token;
        while (iss >> token) {
            tokens.push_back(token);
        }
        return tokens;
    }
};
