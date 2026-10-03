#pragma once
#include "screen.h"
bool ep_display_recovery_required(void);
ep_sink ep_display_guard(ep_sink);
// Configure before use; region callbacks share the wrapped full sink context.
ep_region_sink ep_display_guard_regions(ep_region_sink);
