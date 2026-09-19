1. `cacheWarm` — it ignores `store.Load` errors with `continue`; downstream, `readIndex` then returns `ErrNotFound` for those keys, hiding the real load failure.

2. The default is `3` (`defaultRetryBudget`), set in `newClient` as `retryBudget: defaultRetryBudget`; `drainQueue` can exceed the client’s retry budget because it uses the constant `defaultRetryBudget` per queued key instead of `c.retryBudget`.

3. `lockAll` and `flushSegment` — `lockAll` holds `c.mu` and calls `flushSegment`, which also calls `c.mu.Lock()`; Go’s `sync.Mutex` is not reentrant, so the same goroutine deadlocks.