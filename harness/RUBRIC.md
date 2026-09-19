# Scoring rubric

Quality is the only comparison in this test. Throughput numbers are recorded in
`meta.json` for the record, but the two models run on different silicon and
different runtimes, so no speed claim is made from them.

## Blind procedure

1. `score_prep.py` copies each task's two answers into `blind/<task>/A.md` and
   `blind/<task>/B.md`, with the A/B assignment randomized per task and the
   mapping written to `blind/KEY.json`.
2. The grader reads only `blind/`. Model names appear nowhere in those files.
3. Scores go in `SCORES.csv`. The key is joined afterward.

## Per-task scoring

| Axis | Range | What it measures |
|---|---|---|
| Correctness | 0-5 | Does it do what was asked, and would it run? Compilable code is checked with the real toolchain (see below). |
| Instruction following | 0-3 | Constraints honored exactly: stdlib only, return-only-code, idempotency, named identifiers, output shape. |
| Concision | 0-2 | No unrequested commentary, no restating the prompt, no filler scaffolding. |
| Hallucination | flag | Any invented API, package, tool, column, or flag. One flag per instance, listed by name. |

Maximum 10 per task, 80 for the suite, plus the hallucination count as a separate
figure that is never netted into the score.

## Mechanical checks before human scoring

These are objective and are run on both answers identically, in a throwaway
container, never against a real project:

| Task | Check |
|---|---|
| T01 go-handler | `gofmt -l`, `go vet`, `go build` against a stub `Subscriber` |
| T02 ts-bugfix | `tsc --noEmit` with a stub `next/server` and `@/lib/db` |
| T03 n8n-code-node | `node --check`, then executed against a fixed input array with a frozen clock |
| T04 sql-migration | applied twice against a scratch PostgreSQL 16 container; row counts compared after each run |
| T05 log-diagnosis | no mechanical check; graded on whether the stated cause matches the known one |
| T06 refactor-tests | `go test ./...` on the model's own tests, then the model's tests run against a known-good reference implementation |
| T07 long-context | answers compared to the grading key in `fixtures/gen_longctx.py` |
| T08 tool-call | JSON parses, tool names and required arguments validate against the schema |

A mechanical failure caps Correctness at 2 for that task regardless of how the
prose reads.

## Known asymmetries, stated up front

- Bonsai 2 27B is a ternary quantization of Qwen3.8-27B. This is a quantization
  test, not a contest between two model families.
- Bonsai runs on an RTX 2080 SUPER (8 GB, Turing) under llama.cpp with Q4 KV;
  Qwen3.8-27B runs GPTQ-Int4 on an Intel Arc Pro B70 under vLLM. Different
  quantizer, different runtime, different kernels.
- Q4 KV is a deliberate handicap on the Bonsai side, chosen to fit 8 GB. Any
  degradation it causes is part of what the 8 GB claim actually costs.
- Both models run with thinking enabled. The 98.2% retention figure PrismML
  publishes is measured in thinking mode; testing without it would measure
  something else.
- Sampling is fixed for both: temperature 0.2, top_p 0.95, seed 1729, one
  generation per task. Single-sample results are noisy; that is a limitation of
  this test, not a hidden variable.
