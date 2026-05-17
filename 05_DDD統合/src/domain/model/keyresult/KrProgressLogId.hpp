#pragma once

#include "../../../common/UuidId.hpp"

// =============================================================================
// KrProgressLogId — 進捗ログ（KrProgressLog）IDを表す値オブジェクト
// KeyResult 集約内エンティティの識別子。
// =============================================================================
struct KrProgressLogTag {};
using KrProgressLogId = UuidId<KrProgressLogTag>;
