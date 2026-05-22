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
//   3. PgObjectiveRepository は Period への FK が必要なため接続確認のみ
// =============================================================================

#include <iostream>
#include <memory>
#include <cassert>

#include <pqxx/pqxx>

#include "infrastructure/repository/PgUserRepository.hpp"
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
// テスト 3: PgObjectiveRepository / PgKeyResultRepository — 接続確認のみ
// （Period / Team / User など FK 先が必要なため CRUD は統合テストで行う）
// ─────────────────────────────────────────────────────────────────────────────
void testRepoInstantiation(std::shared_ptr<pqxx::connection> conn) {
    std::cout << "[8] PgObjectiveRepository インスタンス生成... ";
    PgObjectiveRepository objRepo{conn};
    std::cout << "✅\n";

    std::cout << "[9] PgKeyResultRepository インスタンス生成... ";
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
