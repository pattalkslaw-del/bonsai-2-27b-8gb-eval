# Grading notes

Recorded as they happen, during the run, not reconstructed afterward.

## T03 n8n-code-node, qwen38 (Qwen3.8-27B GPTQ-Int4, vLLM on Lawlab: EPYC 7532, 256 GB DDR4, two Arc Pro B70s)

Scored **0 for correctness**. The request returned 32,599 completion tokens and an
empty `content` field: the model reasoned until it stopped and never emitted the
Code node. Not a truncation artifact of the harness; the budget was 60,000 tokens
and it used just over half. A generation that spends 32k tokens and produces no
answer is a failure of the task, and it is scored as one rather than rerun.

## Reasoning traces are asymmetric

llama.cpp returns Bonsai's thinking in `reasoning_content`, so every Bonsai answer
in `runs/bonsai/` carries its full trace. vLLM counts qw38a's reasoning tokens in
`usage` but returns an empty `reasoning_content`, so `runs/qwen38/` has answers
without traces. The token counts for qw38a are therefore real and its traces are
simply not recoverable from this endpoint. Stated here so the published repo is
not read as one model thinking and the other not.

## Thinking dominates both

Bonsai T01: 2,410 characters of answer against 50,588 of thinking. T02: 1,472
against 59,130. Roughly 90% of every generation is reasoning. Any latency figure
taken from these runs is a reasoning-mode figure and nothing else.

## Correction: the trace asymmetry was a harness bug, not a runtime limitation

The vLLM build on lawlab returns the reasoning field as `reasoning`, not
`reasoning_content`. The harness only read the latter, so the first qw38a pass
saved answers without traces. The runner was patched and the full suite rerun
against the same endpoint with `chat_template_kwargs: {"enable_thinking": true}`
sent explicitly. The first pass is kept at `runs/qwen38-notraces/` so the two
passes can be compared rather than the first being quietly discarded.

## Client timeouts on the Bonsai side

T03 and T05 returned no answer on the first Bonsai pass because the harness cut
the connection at 1,800 seconds while the model was still generating. That is a
harness ceiling, not a model failure, and is distinct from the qw38a T03 zero,
where the model itself finished and emitted nothing. Both tasks were rerun with a
7,200-second ceiling and no other change.

## The real cost of the 8 GB footprint

Decode on the RTX 2080 SUPER holds 20.35 tok/s with all 64 layers resident and
Q4 KV at 64k context. Reasoning runs on these tasks ranged from roughly 13k to
over 36k tokens, so a single coding task costs 8 to 35 minutes on this card. The
model fits in 8 GB as advertised; what it costs is time.
