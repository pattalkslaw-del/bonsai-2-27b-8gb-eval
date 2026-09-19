# T07 long-context (qwen38)

finish_reason: stop  elapsed: 137.26s  usage: {"prompt_tokens": 16077, "total_tokens": 24067, "completion_tokens": 7990, "prompt_tokens_details": null}

## Answer



1. `cacheWarm` — it ignores `store.Load` errors with `continue`; downstream, `readIndex` then returns `ErrNotFound` (or missing/stale data) instead of the original load error.

2. The default retry budget is `3`; it is set in `newClient`. `drainQueue` can exceed it because it retries each queued key using `defaultRetryBudget` (ignoring `c.retryBudget`), so total attempts can exceed 3.

3. `lockAll` and `flushSegment` — `lockAll` acquires `c.mu` and then calls `flushSegment`, which also acquires `c.mu`; Go’s `sync.Mutex` is not reentrant, so the same goroutine deadlocks.

