# Findings

Scored run: Bonsai 2 27B PTQ1_0 on an RTX 2080 SUPER (8 GB) vs Qwen3.8-27B GPTQ-Int4 on an Intel Arc Pro B70. Same prompts, same sampling, thinking on. Suite totals: **Bonsai 46 / 80**, **Qwen3.8 71 / 80**.

Speed is not compared. Token counts and elapsed times are in `runs/*/meta.json`.

## Headline

The 8 GB ternary pack loads, holds 64k context with Q4 KV, and decodes at ~20 tok/s on a Turing 2080 SUPER. On this coding battery it keeps long-context recall and the TypeScript debug task, and it can emit a working n8n node. It loses the two longest reasoning jobs (T04, T05) by never leaving the think block, and it corrupts identifiers in Go (`net/mail.ParseAddress` written as a single selector). That last failure is the one ternary quantization would predict: a one-token smash inside an otherwise coherent design.

Qwen3.8 GPTQ-Int4 completed every task with `finish_reason=stop` and passed the PostgreSQL migration twice.

## Footprint (Bonsai, this card)

Weights 5.6 GB. All 64 layers resident. Q4 KV.

- 8,192 context: 6,448 MiB
- 65,536 context: 7,724 MiB (this is what served the suite)
- 81,920 and above: compute-buffer OOM on 8 GB

The 8 GB claim is true at 64k. It is not true at 80k+ with this KV type and this card.

## Termination

Bonsai T04 and T05 each burned 60,000 completion tokens, ~45 minutes, `finish_reason=length`, empty `content`. The traces are 260k and 240k characters of thinking and no SQL / no diagnosis. That is scored as 0, a termination failure, not as a wrong answer.

T03 on the clean pass did stop: 50,742 tokens, 37 minutes, a complete Code node. An earlier discarded pass of T03 also ran to 60k with no answer. Same prompt, same seed, different termination. Single-sample noise is real on this model.

Qwen3.8's first T03 also returned empty content after 32,599 tokens (`stop`, not a cutoff). The second sample produced a complete node. That first empty is in `archive/qwen38-notraces/`.

Thinking dominates both: Bonsai answers are 0.4-7k characters against 20-260k of trace. Qwen3.8 is the same shape. Latency figures from these runs are reasoning-mode figures.

## Per-task

### T01 go-handler (Bonsai 6, Qwen 10)

Bonsai's design is the one the prompt asked for: method on a struct, `Subscriber`, `ErrDuplicate`, 3s timeout from the request context, slog on 500, 201/200/405. It does not compile:

```
./main.go:42:15: undefined: net
./main.go:42:19: multiple-value mail.ParseAddress(email) in single-value context
```

The source is `net/mail.ParseAddress(email)` with `net/mail` imported. The slash is treated as division. This is the mixed-pass failure again, on a fresh sample.

Qwen3.8 uses a regex, `ParseForm` correctly, vet-clean.

### T02 ts-bugfix (10, 10)

Both found the unawaited query, the unreturned 404, the `id!` assertion, missing rejection handling, and the unbounded module cache. Both deleted the cache rather than patching it. Qwen3.8 also set `export const dynamic = 'force-dynamic'`.

### T03 n8n-code-node (8, 10)

Executed against a frozen-clock fixture (`Date.now` pinned, `$input.all()` and `$items` both defined).

Both kept the fresh item and the newer duplicate, skipped bad URL and bad date. Survivors match.

Bonsai reads `$items` rather than `$input.all()`. That is not the documented n8n Code-node API for "Run Once for All Items". Dropped=1 (age only). Qwen3.8 dropped=2 (age plus the discarded duplicate). The prompt names `dropped` without defining whether duplicates count; both readings are defensible. The API mismatch is why Bonsai is 4/5 on correctness and 2/3 on instruction following.

### T04 sql-migration (0, 10)

Bonsai: empty. Qwen3.8: trigger-maintained `amount_paid_cents` (generated columns cannot reference another table), `CREATE INDEX` without a schema-qualified index name, backfill with `NOT EXISTS`, applied twice on Postgres 16: 5 payments, 5 paid invoices reconciled, 1 index, extra stripe insert still reconciles.

The discarded Bonsai pass *did* emit SQL and died on `CREATE INDEX IF NOT EXISTS public.<name>`, which Postgres rejects. That file is in `archive/bonsai-mixed-DISCARD/` and is not scored.

### T05 log-diagnosis (0, 9)

Bonsai: empty. Qwen3.8 names the official image's temporary initdb server, why `pg_isready` is green during that window, and a healthcheck that rejects `-c config_file=` in `postmaster.opts`, plus `start_period`. That is the known cause. Concision 1 because the answer is long.

### T06 refactor-tests (5, 5)

Both fail `go test` on their own tests, so correctness is capped at 2 by the rubric.

Bonsai's implementation applies `if fee > cap { fee = cap }` even when `cap == 0`, so every "no cap" case (cap passed as 0) returns 0. The tests contradict themselves: one case wants 100 with cap 0, another wants 0 with cap 0.

Qwen3.8 fails 1 of ~35 cases (`0.005h` at `2c/h` rounding). Signature was changed to `time.Duration` hours and `*int` cap, which is extra surface relative to the prompt.

### T07 long-context (10, 10)

Planted key (from `fixtures/gen_longctx.py`):

1. `cacheWarm` discards `store.Load` errors; `readIndex` later returns `ErrNotFound`.
2. `defaultRetryBudget = 3`, assigned in `newClient`; `drainQueue` retries in an inner loop that never decrements `c.retryBudget`.
3. `lockAll` and `flushSegment` both take `mu`; `lockAll` holds it while calling `flushSegment`.

Both models answered all three. Qwen3.8's Q2 is the closer match (constant used per key instead of `c.retryBudget`). Bonsai said drainQueue can exceed the budget in aggregate across keys, which is also true of the code.

Prompt tokens: 16,077. Bonsai prefill on this task did not use a cached prefix (`cached_tokens: 0`).

### T08 tool-call (7, 7)

Both returned a 3-call JSON array: `list_matters(status=open)`, one `get_invoice`, `send_email` to `billing@example.com`. JSON parses. No invented tools.

`get_invoice.matter_id` is a single string. Both passed a list template (`{{step_1.matters}}` / `{{step_1.matters[*].id}}`) instead of one call per matter. Neither has a step that filters unpaid invoices over $500; they dumped `{{step_2.invoices}}` into the email body. Same defect on both sides.

## Hallucination flags (not netted into the score)

| model | task | flag |
|---|---|---|
| Bonsai | T03 | `$items` as the n8n all-items collection |
| Qwen3.8 | none listed | |

Bonsai T01's `net/mail.ParseAddress` is identifier corruption, not an invented API; `net/mail.ParseAddress` is real.

## Token / time record (not a speed comparison)

| task | Bonsai s | Bonsai tok | Qwen s | Qwen tok | Bonsai finish |
|---|---|---|---|---|---|
| T01 | 359 | 9,688 | 382 | 24,786 | stop |
| T02 | 493 | 13,155 | 258 | 16,914 | stop |
| T03 | 2,212 | 50,742 | 400 | 25,826 | stop |
| T04 | 2,716 | 60,000 | 319 | 21,010 | length |
| T05 | 2,721 | 60,000 | 506 | 34,162 | length |
| T06 | 1,065 | 26,735 | 551 | 39,318 | stop |
| T07 | 255 | 4,976 | 119 | 7,005 | stop |
| T08 | 244 | 6,660 | 94 | 6,457 | stop |

Bonsai wall time for the scored pass: started 2026-09-18 17:07 CT, finished 19:55 CT, about 2 hours 48 minutes.

## Limitations

- n=1 per task. Bonsai T01 token count moved from 13,219 (discarded pass) to 9,688 (scored) on the same seed.
- Grading was open, not blind.
- Q4 KV is part of the 8 GB setup; FP16 KV was not measured.
- Vision pack was not tested.
- No tool-use runtime; T08 is a JSON plan, not executed tools.
- Qwen T03 in the scored directory is a second sample.
- Different quantizer, runtime, and GPU; only quality is compared.

## Practical reading

If the question is "does the ternary 27B fit and run on an 8 GB Turing card?" yes, at 64k with Q4 KV, at ~20 tok/s decode.

If the question is "does it retain coding quality against GPTQ-Int4 of the same base?" on this battery, no: 46 vs 71, with the gap concentrated on (a) never finishing T04/T05, (b) Go identifier corruption, (c) n8n API slip. Long-context recall and the TypeScript debug did not collapse.
