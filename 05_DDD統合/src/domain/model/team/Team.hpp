#pragma once

#include <string>
#include <vector>
#include <optional>
#include <algorithm>
#include <utility>
#include "TeamId.hpp"
#include "TeamMember.hpp"
#include "../user/UserId.hpp"
#include "../../../common/Result.hpp"

// =============================================================================
// Team — チームエンティティ（集約ルート）
//
// ドメインルール:
//   - name は空にできない（最大 100 文字）
//   - 同一 user を同一 team に重複追加してはならない
//   - admin は team に必ず 1 人以上存在しなければならない（admin 削除時に検証）
//   - TeamMember は Team 集約内でのみ管理する
// =============================================================================
class Team {
public:
    // ファクトリ：新規チーム生成
    static Result<Team> create(
        TeamId                   id,
        std::string              name,
        std::optional<std::string> description = std::nullopt
    ) {
        auto nameResult = validateName(name);
        if (!nameResult) return Result<Team>::err(nameResult.error());

        return Result<Team>::ok(Team{
            std::move(id),
            std::move(name),
            std::move(description)
        });
    }

    // ─── アクセサ ───────────────────────────────────────────
    const TeamId&                       id()          const { return id_; }
    const std::string&                  name()        const { return name_; }
    const std::optional<std::string>&   description() const { return description_; }
    const std::vector<TeamMember>&      members()     const { return members_; }
    const std::string&                  createdAt()   const { return createdAt_; }
    const std::string&                  updatedAt()   const { return updatedAt_; }

    // ─── ドメイン操作 ────────────────────────────────────────
    // チーム名変更
    Result<void> changeName(const std::string& newName) {
        auto result = validateName(newName);
        if (!result) return result;
        name_ = newName;
        return Result<void>::ok();
    }

    // 説明文変更
    void changeDescription(std::optional<std::string> desc) {
        description_ = std::move(desc);
    }

    // メンバー追加（重複チェック付き）
    Result<void> addMember(TeamMember member) {
        if (hasMember(member.userId())) {
            return Result<void>::err(
                "Team: ユーザー " + member.userId().value() + " はすでにチームメンバーです"
            );
        }
        members_.push_back(std::move(member));
        return Result<void>::ok();
    }

    // メンバー削除（admin が 0 人になる場合はエラー）
    Result<void> removeMember(const UserId& userId) {
        auto it = findIterator(userId);
        if (it == members_.end()) {
            return Result<void>::err(
                "Team: ユーザー " + userId.value() + " はこのチームのメンバーではありません"
            );
        }
        // admin を削除する場合、残り admin が 0 人になるか確認
        if (it->role().isAdmin() && adminCount() <= 1) {
            return Result<void>::err("Team: チームには admin が最低 1 人必要です");
        }
        members_.erase(it);
        return Result<void>::ok();
    }

    // メンバーのロール変更
    Result<void> changeMemberRole(const UserId& userId, Role newRole) {
        auto it = findIterator(userId);
        if (it == members_.end()) {
            return Result<void>::err(
                "Team: ユーザー " + userId.value() + " はこのチームのメンバーではありません"
            );
        }
        // admin から member に降格する場合、残り admin が 0 人になるか確認
        if (it->role().isAdmin() && newRole.isMember() && adminCount() <= 1) {
            return Result<void>::err("Team: チームには admin が最低 1 人必要です");
        }
        it->changeRole(std::move(newRole));
        return Result<void>::ok();
    }

    // メンバー検索
    const TeamMember* findMember(const UserId& userId) const {
        auto it = std::find_if(members_.begin(), members_.end(),
            [&userId](const TeamMember& m) { return m.userId() == userId; });
        return it != members_.end() ? &(*it) : nullptr;
    }

    bool hasMember(const UserId& userId) const {
        return findMember(userId) != nullptr;
    }

    // DB再構築用：メンバーリストを一括セット（インフラ層からのみ呼び出す）
    void setMembers(std::vector<TeamMember> members) {
        members_ = std::move(members);
    }

    // DB再構築用：タイムスタンプをセット
    void setTimestamps(std::string createdAt, std::string updatedAt) {
        createdAt_ = std::move(createdAt);
        updatedAt_ = std::move(updatedAt);
    }

    // ─── 等価比較（ID で同一性を判定）───────────────────────
    bool operator==(const Team& other) const { return id_ == other.id_; }
    bool operator!=(const Team& other) const { return !(*this == other); }

private:
    Team(TeamId id, std::string name, std::optional<std::string> description)
        : id_(std::move(id))
        , name_(std::move(name))
        , description_(std::move(description))
    {}

    static Result<void> validateName(const std::string& name) {
        if (name.empty()) {
            return Result<void>::err("Team: チーム名が空です");
        }
        if (name.size() > 100) {
            return Result<void>::err(
                "Team: チーム名が100文字を超えています: " + std::to_string(name.size()) + "文字"
            );
        }
        return Result<void>::ok();
    }

    std::vector<TeamMember>::iterator findIterator(const UserId& userId) {
        return std::find_if(members_.begin(), members_.end(),
            [&userId](const TeamMember& m) { return m.userId() == userId; });
    }

    int adminCount() const {
        return static_cast<int>(std::count_if(members_.begin(), members_.end(),
            [](const TeamMember& m) { return m.role().isAdmin(); }));
    }

    TeamId                      id_;
    std::string                 name_;
    std::optional<std::string>  description_;
    std::vector<TeamMember>     members_;
    std::string                 createdAt_;
    std::string                 updatedAt_;
};
