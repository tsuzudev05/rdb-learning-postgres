// =============================================================================
// main.cpp — PgRepository スモークテスト
//
// DevContainer 内で実行:
//   cd /workspace/05_DDD統合
//   make test
//
// または:
//   bash /workspace/scripts/test.sh
//
// テスト内容:
//   1. PostgreSQL 接続確認
//   2. PgUserRepository::save / findById / findByEmail / findAll / remove
//   3. PgObjectiveRepository / PgKeyResultRepository のインスタンス生成確認
// =============================================================================

#include <iostream>
#include <memory>
#include <cassert>
#include <stdexcept>

#include <pqxx/pqxx>

#include "infrastructure/repository/PgUserRepository.hpp"
#include "infrastructure/repository/PgObjectiveRepository.hpp"
#include "infrastructure/repository/PgKeyResultRepository.hpp"

// 接続文字列（環境変数 DATABASE_URL を優先）
static std::string connectionString() {
    const char* url = std::getenv("DATABASE_URL");
    if (url && url[0] != '\0') return url;
    return "postgresql://postgres:pass@postgres:5432/learning";
}

// ─── ヘルパー ────────────────────────────────────────────────────────────────
template <typename T>
T unwrap(Result<T> result, const std::string& label) {
    if (!result) throw std::runtime_error(label + ": " + result.error());
    return std::move(result.value());
}

void unwrapVoid(Result<void> result, const std::string& label) {
    if (!result) throw std::runtime_error(label + ": " + result.error());
}

// ─── テスト 1: DB 接続確認 ───────────────────────────────────────────────────
void testConnection(pqxx::connection& conn) {
    std::cout << "[1] DB 接続確認... ";
    pqxx::work tx{conn};
    auto r = tx.exec("SELECT current_database()");
    tx.commit();
    std::cout << "✅  DB=" << r[0][0].as<std::string>() << "\n";
}

// ─── テスト 2: PgUserRepository CRUD ────────────────────────────────────────
void testUserRepository(std::shared_ptr<pqxx::connection> conn) {
    PgUserRepository repo{conn};
    const std::string uuid = "11111111-1111-4111-a111-111111111111";

    auto id    = UserId::create(uuid).value();
    auto email = Email::create("smoke@example.com").value();

    // クリーンアップ（冪等）
    unwrapVoid(repo.remove(id), "remove(pre-cleanup)");

    // save（新規）
    std::cout << "[2] save (新規)... ";
    unwrapVoid(repo.save(User::create(id, "スモークテスト太郎", email, "hash_xxx").value()), "save");
    std::cout << "✅\n";

    // findById
    std::cout << "[3] findById... ";
    auto found = unwrap(repo.findById(id), "findById");
    assert(found.has_value() && found->name() == "スモークテスト太郎");
    std::cout << "✅  name=" << found->name() << "\n";

    // save（upsert 更新）
    std::cout << "[4] save (更新)... ";
    unwrapVoid(repo.save(User::create(id, "スモークテスト更新済み", email, "hash_yyy").value()), "save(update)");
    auto updated = unwrap(repo.findById(id), "findById(after update)");
    assert(updated.has_value() && updated->name() == "スモークテスト更新済み");
    std::cout << "✅  name=" << updated->name() << "\n";

    // findByEmail
    std::cout << "[5] findByEmail... ";
    auto byEmail = unwrap(repo.findByEmail(email), "findByEmail");
    assert(byEmail.has_value() && byEmail->id() == id);
    std::cout << "✅\n";

    // findAll
    std::cout << "[6] findAll... ";
    auto all = unwrap(repo.findAll(), "findAll");
    assert(!all.empty());
    std::cout << "✅  count=" << all.size() << "\n";

    // remove
    std::cout << "[7] remove... ";
    unwrapVoid(repo.remove(id), "remove");
    auto gone = unwrap(repo.findById(id), "findById(after remove)");
    assert(!gone.has_value());
    std::cout << "✅\n";
}

// ─── テスト 3: Objective / KeyResult リポジトリ インスタンス生成確認 ─────────
void testRepoInstantiation(std::shared_ptr<pqxx::connection> conn) {
    std::cout << "[8] PgObjectiveRepository 生成... ";
    PgObjectiveRepository objRepo{conn};
    std::cout << "✅\n";

    std::cout << "[9] PgKeyResultRepository 生成... ";
    PgKeyResultRepository krRepo{conn};
    std::cout << "✅\n";
}

// ─── main ────────────────────────────────────────────────────────────────────
int main() {
    std::cout << "========================================\n";
    std::cout << " OKR Repository スモークテスト\n";
    std::cout << "========================================\n";
    try {
        auto conn = std::make_shared<pqxx::connection>(connectionString());
        testConnection(*conn);
        testUserRepository(conn);
        testRepoInstantiation(conn);
        std::cout << "========================================\n";
        std::cout << " ✅ 全テスト PASSED\n";
        std::cout << "========================================\n";
        return 0;
    } catch (const std::exception& e) {
        std::cerr << "\n❌ テスト失敗: " << e.what() << "\n";
        return 1;
    }
}
