#pragma once

#include "../../../common/UuidId.hpp"

// =============================================================================
// PeriodId — 期間（半期 OKRサイクル）IDを表す値オブジェクト
// =============================================================================
struct PeriodTag {};
using PeriodId = UuidId<PeriodTag>;
