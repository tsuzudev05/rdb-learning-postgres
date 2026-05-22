#pragma once

#include "../../../common/UuidId.hpp"

// =============================================================================
// TeamMemberId — チームメンバーIDを表す値オブジェクト
// Team 集約内エンティティ（TeamMember）の識別子。
// =============================================================================
struct TeamMemberTag {};
using TeamMemberId = UuidId<TeamMemberTag>;
