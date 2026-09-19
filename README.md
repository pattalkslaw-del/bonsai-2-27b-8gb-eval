# Ternary Bonsai 2 27B on 8 GB: coding-suite eval

Independent measurement of [PrismML Ternary Bonsai 2 27B](https://huggingface.co/prism-ml/Ternary-Bonsai-2-27B-gguf) (`PTQ1_0`) running in 8 GB VRAM, compared with the same base model served as Qwen3.8-27B GPTQ-Int4.

This is a **quantization and serving test**, not a contest between two model families. Bonsai 2 27B is a ternary pack of Qwen3.8-27B. The two artifacts ran on different silicon and different runtimes; **no speed claim is made from the wall times**.

Date of the scored runs: 2026-09-18 (America/Chicago).

| | Bonsai 2 27B PTQ1_0 | Qwen3.8-27B GPTQ-Int4 |
|---|---|---|
| Suite total (max 80) | **46** | **71** |
| Empty answers | T04, T05 (both `finish_reason=length` at 60k) | none in the scored set |
| Compile/apply failures | T01 does not compile | none |

Copy-ready wiki page: [WIKI.md](WIKI.md). Expanded write-up: [FINDINGS.md](FINDINGS.md). Table: [SCORES.csv](SCORES.csv). Serve flags: [SETTINGS.md](SETTINGS.md). Protocol: [METHOD.md](METHOD.md).

## What is in this repo

```
README.md          this file
FINDINGS.md        results, per-task notes, limitations
SCORES.csv         correctness / instruction / concision
METHOD.md          rubric, sampling, what is and is not claimed
SETTINGS.md        model files, llama.cpp fork, build, serve flags, VRAM ceiling
PROCESS-NOTES.md   what was discarded and why
harness/           prompts, runner, rubric, mechanical checker
fixtures/          long-context Go file and its generator
runs/bonsai/       scored Bonsai outputs (answer + full thinking trace)
runs/qwen38/       scored Qwen3.8 outputs
archive/           mixed/discarded Bonsai pass; first Qwen3.8 pass without traces
checks/            mechanical-check results and extracted sources
```

Weights are **not** in this repo. Pull them from Hugging Face. The PrismML llama.cpp fork is **not** vendored; clone it from upstream (commit pinned in SETTINGS.md).

## Reproduce the suite

1. Serve Bonsai as in SETTINGS.md (PrismML `prism` branch, `CUDA_DEVICE_ORDER=PCI_BUS_ID`).
2. Serve Qwen3.8-27B GPTQ-Int4 however you like; the client is OpenAI-compatible `/v1/chat/completions`.
3. From `harness/`:

```bash
python3 run_suite.py --label bonsai --base-url http://127.0.0.1:8097/v1 \
    --model bonsai2-27b-ptq1_0 --out ../runs/bonsai --timeout 7200

python3 run_suite.py --label qwen38 --base-url http://127.0.0.1:8000/v1 \
    --model qwen38 --out ../runs/qwen38 --timeout 7200
```

Sampling is fixed: temperature 0.2, top_p 0.95, seed 1729, thinking on, one sample per task. `max_tokens` is 60000 except T07 (45000).

A single coding task on the 2080 SUPER took 6 to 45 minutes because reasoning consumed 10k to 60k tokens at ~20 tok/s. Plan for that. Do not change the request body mid-suite.

## Hardware used here

- Bonsai: Render-1, RTX 2080 SUPER 8 GB (Turing, sm_75), PrismML llama.cpp, Q4 KV, 65536 context, all 64 layers on GPU.
- Qwen3.8: lawlab, Intel Arc Pro B70, vLLM GPTQ-Int4, 131072 max length.

## License

MIT for the harness, prompts, scores, and write-up. Model weights and the PrismML fork keep their upstream licenses.
