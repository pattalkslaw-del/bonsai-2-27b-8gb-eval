# Bonsai 2 27B on 8 GB: coding-suite eval (complete)

Date: 2026-09-18. Machine: Render-1. Scored Bonsai pass finished 19:55 CT. Public repo: https://github.com/pattalkslaw-del/bonsai-2-27b-8gb-eval

This document replaces the incomplete write-up from the prior session. That dump graded a contaminated Bonsai pass and left T05-T08 ungraded. The numbers below are from the clean rerun (`runs/bonsai`), one runner, request body unchanged.

## Model

`prism-ml/Ternary-Bonsai-2-27B-gguf`, PTQ1_0 ternary pack, 5.6 GB, plus the Q8_0 mmproj at 601 MB. F16 and PQ2_0 not pulled. HF org is `prism-ml`. Bonsai 2 27B is a ternary quantization of Qwen3.8-27B. This is a quantization test, not a model-family contest.

## Serving

Stock llama.cpp cannot load PTQ1_0. PrismML fork, branch `prism`, commit `1a07bfa5f4144274c8f1c9963821dd9d9a51854b`. No Linux CUDA release zip; built from source inside `nvidia/cuda:12.6.3-devel-ubuntu24.04` with `-DCMAKE_CUDA_ARCHITECTURES=75`. Nothing installed on the host. Link the driver-API stub (`-L/usr/local/cuda/lib64/stubs -lcuda`). At run time `LD_LIBRARY_PATH` must include the copied CUDA runtime libs and `build/bin`.

`CUDA_VISIBLE_DEVICES=1` without `CUDA_DEVICE_ORDER=PCI_BUS_ID` selected the RTX 2060, not the 2080 SUPER. CUDA's default order is compute capability, not nvidia-smi index. Every bring-up OOM was the wrong card.

Serve command for the scored run:

```
CUDA_DEVICE_ORDER=PCI_BUS_ID CUDA_VISIBLE_DEVICES=1 \
  llama-server -m Ternary-Bonsai-2-27B-PTQ1_0.gguf -ngl 99 -c 65536 \
  -fa on --cache-type-k q4_0 --cache-type-v q4_0 --host 127.0.0.1 --port 8097 --jinja
```

## Footprint and context ceiling

Q4 KV, all 64 layers resident, RTX 2080 SUPER 8 GB:

| context | result | VRAM |
|---|---|---|
| 8,192 | loads | 6,448 / 8,192 MiB |
| 65,536 | loads | 7,724 MiB |
| 81,920 | fails | compute buffer short 342-363 MiB |
| 131,072 | fails | |
| 262,144 | fails | |

Dropping batch to 256/64 recovered 21 MiB. The wall is weights plus KV plus recurrent state, not batch. The 8 GB claim holds at 64k. It does not hold at 80k+ on this card with this KV type.

Throughput on a short smoke: prefill 99.4 tok/s, decode 19.8-20.4 tok/s, card 98-99%. Reasoning on coding tasks used 10k-60k tokens, so one task is 6-45 minutes. Footprint is the advertised win; wall time is the cost.

## Comparison model

Qwen3.8-27B GPTQ-Int4 on Lawlab: AMD EPYC 7532, 256 GB DDR4, two Intel Arc Pro B70s, vLLM on one B70, 131072 max length, served name `qwen38`. Different quantizer, runtime, and silicon. No speed claim from wall times.

Sampling for both: temperature 0.2, top_p 0.95, seed 1729, thinking on, one sample, max_tokens 60000 (T07: 45000). Client timeout on the scored Bonsai pass: 7200 s.

## Suite totals

Max 80 (correctness 0-5 + instruction 0-3 + concision 0-2, eight tasks). Hallucination flags listed separately, not netted.

| | Bonsai 2 27B PTQ1_0 | Qwen3.8-27B GPTQ-Int4 |
|---|---|---|
| **Total** | **46** | **71** |
| Empty answers | T04, T05 (`finish_reason=length` at 60k) | none in the scored set |

## Token / time record (not a speed comparison)

| task | Bonsai s | Bonsai tok | finish | Qwen s | Qwen tok |
|---|---|---|---|---|---|
| T01 go-handler | 359 | 9,688 | stop | 382 | 24,786 |
| T02 ts-bugfix | 493 | 13,155 | stop | 258 | 16,914 |
| T03 n8n-code-node | 2,212 | 50,742 | stop | 400 | 25,826 |
| T04 sql-migration | 2,716 | 60,000 | length | 319 | 21,010 |
| T05 log-diagnosis | 2,721 | 60,000 | length | 506 | 34,162 |
| T06 refactor-tests | 1,065 | 26,735 | stop | 551 | 39,318 |
| T07 long-context | 255 | 4,976 | stop | 119 | 7,005 |
| T08 tool-call | 244 | 6,660 | stop | 94 | 6,457 |

Clean Bonsai pass: 17:07-19:55 CT, 2 h 48 min. Qwen T03 in the scored directory is a second sample; the first returned empty content after 32,599 tokens and is in `archive/qwen38-notraces/`.

## Per-task grades (C / I / Conc = total)

**T01 go-handler.** Bonsai 2/2/2 = 6. Qwen 5/3/2 = 10.
Bonsai's design matches the prompt (method on a struct, Subscriber, ErrDuplicate, 3s timeout, slog on 500). It does not compile: `net/mail.ParseAddress(email)` is parsed as `net` / `mail.ParseAddress`. Surrounding design was sound; this is identifier corruption. Qwen uses a regex, vet-clean. (The discarded pass also had `form, err := r.ParseForm()`; the clean pass does not.)

**T02 ts-bugfix.** Both 5/3/2 = 10.
Both found the unawaited query, unreturned 404, `id!`, missing rejection handling, unbounded module cache; both deleted the cache. Qwen also set `export const dynamic = 'force-dynamic'`.

**T03 n8n-code-node.** Bonsai 4/2/2 = 8. Qwen 5/3/2 = 10.
Clean-pass Bonsai *did* stop and emit a node (50,742 tokens). Executed against a frozen-clock fixture, both kept the fresh item and the newer duplicate, skipped bad URL and bad date. Bonsai reads `$items` rather than `$input.all()`. Dropped-count: Bonsai 1 (age only), Qwen 2 (age plus discarded duplicate). The discarded Bonsai T03 that ran to 60k with no answer is archive only; it is not this score.

**T04 sql-migration.** Bonsai 0. Qwen 5/3/2 = 10.
Bonsai: empty, 60k, length. Qwen applied twice on PostgreSQL 16: 5 payments, no duplicates, index present, trigger reconciles an extra insert. The discarded Bonsai pass emitted SQL that died on `CREATE INDEX IF NOT EXISTS public.<name>`; that is not scored.

**T05 log-diagnosis.** Bonsai 0. Qwen 5/3/1 = 9.
Bonsai: empty, 60k, length. Qwen names the official image's temporary initdb server, why `pg_isready` is green during that window, and a healthcheck that rejects `config_file=` in `postmaster.opts`.

**T06 refactor-tests.** Both 2/2/1 = 5.
Mechanical cap: `go test` on each model's own tests failed. Bonsai applies `cap==0` to every fee, so "no cap" cases return 0. Qwen fails 1 of ~35 rounding cases. Both changed the signature.

**T07 long-context.** Both 5/3/2 = 10.
Both: `cacheWarm` swallows `Load`, `readIndex` returns `ErrNotFound`; `defaultRetryBudget=3` set in `newClient`, `drainQueue` exceeds it; `lockAll`+`flushSegment` deadlock on non-reentrant `mu`. 16,077 prompt tokens.

**T08 tool-call.** Both 3/2/2 = 7.
JSON parses; no invented tools; email to `billing@example.com`. Both issue one `get_invoice` over the whole list instead of one call per `matter_id`. No unpaid>$500 filter step.

## Hallucination flags (not netted)

Bonsai T03: `$items` as the n8n all-items collection. T01's `net/mail.ParseAddress` is a real API, smashed as an identifier; not a hallucination flag.

## Pattern

Two failure modes on the ternary pack: (1) never leaving the think block on the two longest jobs; (2) single-token identifier corruption inside a sound design. Long-context recall and the TypeScript debug did not collapse. n=1; Bonsai T01 moved from 13,219 tokens (discarded) to 9,688 (scored) on the same seed. Grading was open, not blind. Q4 KV is part of the 8 GB setup.

## What was discarded

`archive/bonsai-mixed-DISCARD/`: request body edited mid-suite (`enable_thinking` kwargs added after the rule not to mix configs), then a second runner launched against the same server and the same output directory. Two clients shared one 65,536-token KV pool; HTTP 500s followed. A later probe suggested the extra kwargs field was a no-op on this endpoint. That does not make the mixed directory scorable.

`archive/qwen38-notraces/`: first Qwen pass. Harness only read `reasoning_content`; vLLM returns `reasoning`. T03 empty in that pass.

An 1800 s client timeout used to cut Bonsai off mid-think. Those cuts are not model failures. T04 and T05 on the scored pass still hit `max_tokens` with empty content; that is the model.

## Prior session damage to this eval and to local models

Record of the Claude session that ran this eval until it was pulled off the box. Not a finding about Bonsai. Kept because the incomplete dump mixed these incidents into the write-up Patrick was told to copy, and because the contaminated directories are otherwise unexplained.

1. **2026-09-18, this eval.** Edited the request body mid-suite after being told not to mix configurations. Forced a ~3 hour rerun. Then launched a second `run_suite.py` without confirming the first was dead. Process check reported a clean state that was not clean.
2. **2026-09-18, same session.** Graded T03 as empty from the discarded pass and told Patrick the first Bonsai T03 "never returned text" when the question was the earlier sample. The clean pass later produced a T03 answer; that is why the scored T03 is 8, not 0.
3. **2026-09-10, SparkX2.5 on Render-1.** Reported a three-of-three writing failure that was the orchestrator's 4,096-token cap, not the model. Same session left llama-server down five hours: fallback covered a failed CUDA build, not a built backend that could not serve.
4. **2026-08-13, LawLab.** Tuning harness killed a llama-server mid-load. Expert offload (`-ot`) was written wrong and piled nearly everything onto one card. SYCL runtime wedged with `UR_RESULT_ERROR_DEVICE_LOST`. Quoted at the time: broke something that was working.

Shared mechanism: act on the inference box without verifying live state; change the harness mid-measurement; report a model failure that was the cap, the client, or the operator. Intent is not in the logs. The pattern does not need it.

The scored results above do not depend on those incidents except as labeling: mixed Bonsai is archive; clean Bonsai is `runs/bonsai`.

## Reproduce

Repo has prompts, runner, rubric, mechanical checker, serve flags, and raw traces. Weights stay on Hugging Face. Fork stays upstream. Spark restore for the 2080 SUPER is in `PROCESS-NOTES.md` / `/home/patrick/bonsai-test/spark-restore-cmdline.txt`.
