// =============================================================================
// main.cpp — PgRepository スモークテスト
//
// DevContainer 内で以下のコマンドで実行:
//   cd /workspace/05_DDD統合
//   make run
//
// または:
//   ./scripts/build.sh run
//
// テスト内容:
//   1. PostgreSQL 接続確認
//   2. PgUserRepository::save / findById / findAll / remove
//   3. PgTeamRepository::save / findById / findByUserId / findAll / remove
//   4. PgPeriodRepository::save / findById / findByTeamId / findAll / remove
//   5. PgObjectiveRepository / PgKeyResultRepository — インスタンス生成確認
// =============================================================================

#include <iostream>
#include <memory>
#include <cassert>

#include <pqxx/pqxx>

#include "infrastructure/repository/PgUserRepository.hpp"
#include "infrastructure/repository/PgTeamRepository.hpp"
#include "infrastructure/repository/PgPeriodRepository.hpp"
#include "infrastructure/repository/PgObjectiveRepository.hpp"
#include "infrastructure/repository/PgKeyResultRepository.hpp"

// 接続文字列（環境変数 DATABASE_URL を優先、なければデフォルト値）
static std::string connectionString() {
    const char* url = std::getenv("DATABASE_URL");
    if (url && url[0] != '\0') return url;
    return "postgresql://postgres:pass@postgres:5432/learning";
}

// ─────────────────────────────────────────────────────────────────────────────
// ヘルパー：Result をチェックして失敗時は例外を投げる
// ─────────────────────────────────────────────────────────────────────────────
template <typename T>
T unwrap(Result<T> result, const std::string& label) {
    if (!result) {
        throw std::runtime_error(label + " failed: " + result.error());
    }
    return std::move(result.value());
}

void unwrapVoid(Result<void> result, const std::string& label) {
    if (!result) {
        throw std::runtime_error(label + " failed: " + result.error());
    }
}

// ─────────────────────────────────────────────────────────────────────────────
// テスト 1: DB 接続確認
// ─────────────────────────────────────────────────────────────────────────────
void testConnection(pqxx::connection& conn) {
    std::cout << "[1] DB 接続確認... ";
    pqxx::work tx{conn};
    pqxx::result r = tx.exec("SELECT current_database(), version()");
    tx.commit();
    std::cout << "✅  DB=" << r[0][0].as<std::string>() << "\n";
}

// ─────────────────────────────────────────────────────────────────────────────
// テスト 2: PgUserRepository CRUD
// ─────────────────────────────────────────────────────────────────────────────
void testUserRepository(std::shared_ptr<pqxx::connection> conn) {
    PgUserRepository repo{conn};
    const std::string uuid = "11111111-1111-4111-a111-111111111111";

    // --- クリーンアップ（冪等にするため先に削除）---
    auto idResult = UserId::create(uuid);
    assert(idResult && "UserId::create failed");
    unwrapVoid(repo.remove(idResult.value()), "remove (pre-cleanup)");

    // --- save（新規）---
    std::cout << "[2] UserRepository::save (新規)... ";
    auto emailResult = Email::create("smoke@example.com");
    assert(emailResult && "Email::create failed");
    auto userResult = User::create(
        idResult.value(), "スモークテスト太郎",
        emailResult.value(), "hashed_password_xxx"
    );
    assert(userResult && "User::create failed");
    unwrapVoid(repo.save(userResult.value()), "save");
    std::cout << "✅\n";

    // --- findById ---
    std::cout << "[3] UserRepository::findById... ";
    auto found = unwrap(repo.findById(idResult.value()), "findById");
    assert(found.has_value() && "User should exist");
    assert(found->name() == "スモークテスト太郎");
    assert(found->email().value() == "smoke@example.com");
    std::cout << "✅  name=" << found->name() << "\n";

    // --- save（更新 upsert）---
    std::cout << "[4] UserRepository::save (更新 upsert)... ";
    auto updatedUserResult = User::create(
        idResult.value(), "スモークテスト更新済み",
        emailResult.value(), "hashed_password_yyy"
    );
    assert(updatedUserResult && "User::create (update) failed");
    unwrapVoid(repo.save(updatedUserResult.value()), "save (update)");
    auto updated = unwrap(repo.findById(idResult.value()), "findById after update");
    assert(updated.has_value() && updated->name() == "スモークテスト更新済み");
    std::cout << "✅  updated name=" << updated->name() << "\n";

    // --- findByEmail ---
    std::cout << "[5] UserRepository::findByEmail... ";
    auto byEmail = unwrap(repo.findByEmail(emailResult.value()), "findByEmail");
    assert(byEmail.has_value() && byEmail->id() == idResult.value());
    std::cout << "✅\n";

    // --- findAll ---
    std::cout << "[6] UserRepository::findAll... ";
    auto all = unwrap(repo.findAll(), "findAll");
    assert(!all.empty() && "findAll should return at least 1 user");
    std::cout << "✅  count=" << all.size() << "\n";

    // --- remove ---
    std::cout << "[7] UserRepository::remove... ";
    unwrapVoid(repo.remove(idResult.value()), "remove");
    auto afterRemove = unwrap(repo.findById(idResult.value()), "findById after remove");
    assert(!afterRemove.has_value() && "User should be deleted");
    std::cout << "✅\n";
}

// ─────────────────────────────────────────────────────────────────────────────
// テスト 3: PgTeamRepository CRUD
// ─────────────────────────────────────────────────────────────────────────────
void testTeamRepository(std::shared_ptr<pqxx::connection> conn) {
    PgUserRepository userRepo{conn};
    PgTeamRepository teamRepo{conn};

    const std::string userUuid = "22222222-2222-4222-a222-222222222222";
    const std::string teamUuid = "33333333-3333-4333-a333-333333333333";
    const std::string memberUuid = "44444444-4444-4444-a444-444444444444";

    // --- クリーンアップ ---
    auto teamIdResult = TeamId::create(teamUuid);
    assert(teamIdResult && "TeamId::create failed");
    unwrapVoid(teamRepo.remove(teamIdResult.value()), "team remove (pre-cleanup)");

    auto userIdResult = UserId::create(userUuid);
    assert(userIdResult && "UserId::create (team test) failed");
    unwrapVoid(userRepo.remove(userIdResult.value()), "user remove (pre-cleanup)");

    // --- テスト用 User を作成（TeamMember の FK 先）---
    auto emailResult = Email::create("team-smoke@example.com");
    assert(emailResult && "Email::create failed");
    auto userResult = User::create(
        userIdResult.value(), "チームテストユーザー",
        emailResult.value(), "hashed_pass"
    );
    assert(userResult && "User::create failed");
    unwrapVoid(userRepo.save(userResult.value()), "user save for team test");

    // --- Team 生成（メンバー付き）---
    std::cout << "[8] TeamRepository::save (新規 + メンバー付き)... ";
    auto teamResult = Team::create(teamIdResult.value(), "スモークチーム", std::string{"テスト用チーム"});
    assert(teamResult && "Team::create failed");

    auto memberIdResult = TeamMemberId::create(memberUuid);
    assert(memberIdResult && "TeamMemberId::create failed");
    auto memberTeamIdResult = TeamId::create(teamUuid);
    assert(memberTeamIdResult);
    auto memberUserIdResult = UserId::create(userUuid);
    assert(memberUserIdResult);
    auto memberResult = TeamMember::create(
        memberIdResult.value(),
        memberTeamIdResult.value(),
        memberUserIdResult.value(),
        Role::admin()
    );
    assert(memberResult && "TeamMember::create failed");
    auto addResult = teamResult.value().addMember(std::move(memberResult.value()));
    assert(addResult && "addMember failed");

    unwrapVoid(teamRepo.save(teamResult.value()), "team save");
    std::cout << "✅\n";

    // --- findById ---
    std::cout << "[9] TeamRepository::findById... ";
    auto found = unwrap(teamRepo.findById(teamIdResult.value()), "team findById");
    assert(found.has_value() && "Team should exist");
    assert(found->name() == "スモークチーム");
    assert(found->members().size() == 1);
    std::cout << "✅  name=" << found->name() << "  members=" << found->members().size() << "\n";

    // --- findByUserId ---
    std::cout << "[10] TeamRepository::findByUserId... ";
    auto byUser = unwrap(teamRepo.findByUserId(userIdResult.value()), "team findByUserId");
    assert(!byUser.empty() && "findByUserId should return at least 1 team");
    std::cout << "✅  count=" << byUser.size() << "\n";

    // --- findAll ---
    std::cout << "[11] TeamRepository::findAll... ";
    auto allTeams = unwrap(teamRepo.findAll(), "team findAll");
    assert(!allTeams.empty());
    std::cout << "✅  count=" << allTeams.size() << "\n";

    // --- remove（Team を削除すると team_members も CASCADE 削除）---
    std::cout << "[12] TeamRepository::remove... ";
    unwrapVoid(teamRepo.remove(teamIdResult.value()), "team remove");
    auto afterRemove = unwrap(teamRepo.findById(teamIdResult.value()), "team findById after remove");
    assert(!afterRemove.has_value() && "Team should be deleted");
    std::cout << "✅\n";

    // --- テスト用 User を後片付け ---
    unwrapVoid(userRepo.remove(userIdResult.value()), "user remove (post-cleanup)");
}

// ─────────────────────────────────────────────────────────────────────────────
// テスト 4: PgPeriodRepository CRUD
// ─────────────────────────────────────────────────────────────────────────────
void testPeriodRepository(std::shared_ptr<pqxx::connection> conn) {
    PgUserRepository   userRepo{conn};
    PgTeamRepository   teamRepo{conn};
    PgPeriodRepository periodRepo{conn};

    const std::string userUuid   = "55555555-5555-4555-a555-555555555555";
    const std::string teamUuid   = "66666666-6666-4666-a666-666666666666";
    const std::string memberUuid = "77777777-7777-4777-a777-777777777777";
    const std::string periodUuid = "88888888-8888-4888-a888-888888888888";

    // --- クリーンアップ ---
    auto periodIdResult = PeriodId::create(periodUuid);
    assert(periodIdResult && "PeriodId::create failed");
    unwrapVoid(periodRepo.remove(periodIdResult.value()), "period remove (pre-cleanup)");

    auto teamIdResult = TeamId::create(teamUuid);
    assert(teamIdResult);
    unwrapVoid(teamRepo.remove(teamIdResult.value()), "team remove (pre-cleanup for period)");

    auto userIdResult = UserId::create(userUuid);
    assert(userIdResult);
    unwrapVoid(userRepo.remove(userIdResult.value()), "user remove (pre-cleanup for period)");

    // --- テスト用 User + Team を作成（Period の FK 先）---
    auto emailResult = Email::create("period-smoke@example.com");
    assert(emailResult);
    auto userResult = User::create(userIdResult.value(), "期間テストユーザー", emailResult.value(), "hash");
    assert(userResult);
    unwrapVoid(userRepo.save(userResult.value()), "user save for period test");

    auto teamResult = Team::create(teamIdResult.value(), "期間テストチーム");
    assert(teamResult);
    auto memberIdResult = TeamMemberId::create(memberUuid);
    assert(memberIdResult);
    auto memberTeamIdR = TeamId::create(teamUuid); assert(memberTeamIdR);
    auto memberUserIdR = UserId::create(userUuid); assert(memberUserIdR);
    auto memberResult  = TeamMember::create(memberIdResult.value(), memberTeamIdR.value(), memberUserIdR.value(), Role::admin());
    assert(memberResult);
    auto addR = teamResult.value().addMember(std::move(memberResult.value()));
    assert(addR);
    unwrapVoid(teamRepo.save(teamResult.value()), "team save for period test");

    // --- Period 生成 ---
    std::cout << "[13] PeriodRepository::save (新規)... ";
    auto halfResult = Half::create("H1");
    assert(halfResult);
    auto drResult = DateRange::create("2025-01-01", "2025-06-30");
    assert(drResult);
    auto periodResult = Period::create(
        periodIdResult.value(),
        teamIdResult.value(),
        "2025-H1",
        halfResult.value(),
        drResult.value()
    );
    assert(periodResult);
    unwrapVoid(periodRepo.save(periodResult.value()), "period save");
    std::cout << "✅\n";

    // --- findById ---
    std::cout << "[14] PeriodRepository::findById... ";
    auto found = unwrap(periodRepo.findById(periodIdResult.value()), "period findById");
    assert(found.has_value());
    assert(found->name() == "2025-H1");
    assert(found->half().toString() == "H1");
    std::cout << "✅  name=" << found->name() << "\n";

    // --- findByTeamId ---
    std::cout << "[15] PeriodRepository::findByTeamId... ";
    auto byTeam = unwrap(periodRepo.findByTeamId(teamIdResult.value()), "period findByTeamId");
    assert(!byTeam.empty());
    std::cout << "✅  count=" << byTeam.size() << "\n";

    // --- findAll ---
    std::cout << "[16] PeriodRepository::findAll... ";
    auto allPeriods = unwrap(periodRepo.findAll(), "period findAll");
    assert(!allPeriods.empty());
    std::cout << "✅  count=" << allPeriods.size() << "\n";

    // --- remove ---
    std::cout << "[17] PeriodRepository::remove... ";
    unwrapVoid(periodRepo.remove(periodIdResult.value()), "period remove");
    auto afterRemove = unwrap(periodRepo.findById(periodIdResult.value()), "period findById after remove");
    assert(!afterRemove.has_value());
    std::cout << "✅\n";

    // --- 後片付け ---
    unwrapVoid(teamRepo.remove(teamIdResult.value()), "team remove (post-cleanup)");
    unwrapVoid(userRepo.remove(userIdResult.value()), "user remove (post-cleanup)");
}

// ─────────────────────────────────────────────────────────────────────────────
// テスト 5: PgObjectiveRepository / PgKeyResultRepository — 接続確認のみ
// （Period / Team / User など FK 先が必要なため CRUD は統合テストで行う）
// ─────────────────────────────────────────────────────────────────────────────
void testRepoInstantiation(std::shared_ptr<pqxx::connection> conn) {
    std::cout << "[18] PgObjectiveRepository インスタンス生成... ";
    PgObjectiveRepository objRepo{conn};
    std::cout << "✅\n";

    std::cout << "[19] PgKeyResultRepository インスタンス生成... ";
    PgKeyResultRepository krRepo{conn};
    std::cout << "✅\n";
}

// ─────────────────────────────────────────────────────────────────────────────
// main
// ─────────────────────────────────────────────────────────────────────────────
int main() {
    std::cout << "========================================\n";
    std::cout << " OKR Repository スモークテスト\n";
    std::cout << "========================================\n";

    try {
        auto conn = std::make_shared<pqxx::connection>(connectionString());

        testConnection(*conn);
        testUserRepository(conn);
        testTeamRepository(conn);
        testPeriodRepository(conn);
        testRepoInstantiation(conn);

        std::cout << "\n========================================\n";
        std::cout << " ✅ 全テスト PASSED\n";
        std::cout << "========================================\n";
        return 0;

    } catch (const std::exception& e) {
        std::cerr << "\n❌ テスト失敗: " << e.what() << "\n";
        return 1;
    }
}
