1. `cacheWarm` silently swallows `Load` errors; failed keys are omitted from `c.index`, so `readIndex` later returns `ErrNotFound` instead of the original error.

2. The default retry budget is `3`; it is set by `defaultRetryBudget` and assigned in `newClient`. `drainQueue` can exceed it in aggregate because it allows up to 3 attempts for each queued key.

3. `lockAll` and `flushSegment` would deadlock in the same goroutine because `lockAll` holds `c.mu` while calling `flushSegment`, which tries to lock the same non-recursive mutex.