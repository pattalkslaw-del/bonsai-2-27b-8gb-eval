# Serving and build settings

Pinned so someone else can load the same artifact the same way.

## Model files

Hugging Face: [`prism-ml/Ternary-Bonsai-2-27B-gguf`](https://huggingface.co/prism-ml/Ternary-Bonsai-2-27B-gguf)

Pulled 2026-09-18:

| file | size |
|---|---|
| `Ternary-Bonsai-2-27B-PTQ1_0.gguf` | 5.6 GB |
| `Ternary-Bonsai-2-27B-mmproj-Q8_0.gguf` | 601 MB |
| `README.md` | 23 KB |

F16 (53.8 GB) and PQ2_0 (7.2 GB) were not pulled. Vision was not tested; the mmproj is on disk only.

The Hugging Face org is `prism-ml`.

## llama.cpp fork

Stock llama.cpp cannot load `PTQ1_0`. Use the PrismML fork:

```
https://github.com/PrismML-Eng/llama.cpp.git
branch: prism
commit: 1a07bfa5f4144274c8f1c9963821dd9d9a51854b
message: Merge pull request #179 from PrismML-Eng/feat/dspark-shared-head-runtime
date: 2026-09-17 23:58:45 -0700
```

No Linux CUDA release zip was published at the time of this run (Windows CUDA zips existed). The binary was built from source.

## Build (CUDA 12.6, sm_75, no host CUDA toolkit)

Render-1 had NVIDIA driver + `nvidia-container-toolkit`, no CUDA toolkit, and sudo requires a password. The build ran inside `nvidia/cuda:12.6.3-devel-ubuntu24.04` and installed nothing on the host. Two details that were load-bearing:

1. Link the driver-API stub: `-L/usr/local/cuda/lib64/stubs -lcuda`.
2. At run time, `LD_LIBRARY_PATH` must include both the copied CUDA runtime libs (`libcudart`, `libcublas`, `libcublasLt`, `libnccl`) and `build/bin` (the fork emits extra `.so` files such as `libllama-server-impl.so`).

Architecture flag: `-DCMAKE_CUDA_ARCHITECTURES=75` (RTX 2080 SUPER / 2060).

Reported version string of the resulting `llama-server`:

```
version: 0.2.0-dev (build 0, commit unknown)
built with GNU 13.2.0 for Linux x86_64
```

## Device selection gotcha

`CUDA_VISIBLE_DEVICES=1` without `CUDA_DEVICE_ORDER=PCI_BUS_ID` selected the RTX 2060, not the 2080 SUPER. CUDA's default order is compute capability, not `nvidia-smi` index. Every OOM during bring-up was the 6 GB card. Always set:

```bash
export CUDA_DEVICE_ORDER=PCI_BUS_ID
export CUDA_VISIBLE_DEVICES=1   # 2080 SUPER on this machine
```

Confirm with `nvidia-smi` while the process is alive.

## Serve command used for the scored Bonsai run

```bash
export LD_LIBRARY_PATH=/path/to/cudalibs:/path/to/build/bin
export CUDA_DEVICE_ORDER=PCI_BUS_ID
export CUDA_VISIBLE_DEVICES=1

./build/bin/llama-server \
  -m /path/to/Ternary-Bonsai-2-27B-PTQ1_0.gguf \
  -ngl 99 \
  -c 65536 \
  -fa on \
  --cache-type-k q4_0 \
  --cache-type-v q4_0 \
  --host 127.0.0.1 \
  --port 8097 \
  --jinja
```

Q4 KV is a deliberate handicap so 64k context fits in 8 GB. Any quality loss it causes is part of what the 8 GB claim costs.

`--jinja` is required for the Qwen chat template.

## Context / VRAM ceiling (this card, this quant, Q4 KV, ngl 99)

| context | result | VRAM |
|---|---|---|
| 8,192 | loads | 6,448 / 8,192 MiB |
| 65,536 | loads | 7,724 MiB |
| 81,920 | fails | compute buffer short ~342-363 MiB |
| 131,072 | fails | |
| 262,144 | fails | |

Dropping batch to 256/64 recovered ~21 MiB; the wall is weights + KV + recurrent state, not batch. The 8 GB claim holds, and 64k holds with it.

## Throughput on the 2080 SUPER (scored server)

From `llama-server` slot timings on a short smoke prompt:

- prefill ~99 tok/s
- decode ~19.8 to 20.4 tok/s, steady, card ~98-99%

Reasoning on the coding tasks used 10k to 60k completion tokens, so one task is 6 to 45 minutes on this card. The footprint is the advertised win; wall time is the cost.

## Comparison endpoint (not interchangeable silicon)

Lawlab is an AMD EPYC 7532 with 256 GB DDR4 and two Intel Arc Pro B70s. Qwen3.8-27B GPTQ-Int4 ran under vLLM on one of those B70s, OpenAI-compatible at `http://<lawlab>:8000/v1`, served name `qwen38`, max length 131072. Traces land in `message.reasoning` on that build, not `reasoning_content`. The harness reads both.

## Client request body (do not change mid-suite)

```json
{
  "model": "<served name>",
  "messages": [
    {"role": "system", "content": "You are a senior software engineer. Answer the task exactly as asked. Do not add commentary the task did not request."},
    {"role": "user", "content": "<prompt>"}
  ],
  "max_tokens": 60000,
  "temperature": 0.2,
  "top_p": 0.95,
  "seed": 1729,
  "stream": false
}
```

T07 uses `max_tokens` 45000. Thinking is left at the server/template default (on). A mid-suite experiment that added `chat_template_kwargs: {"enable_thinking": true}` is archived, not scored; see PROCESS-NOTES.md.

## Ornith-1.5-9B Q4_K_M (addendum, same card)

Hugging Face: [`ornith-ai/Ornith-1.5-9B-GGUF`](https://huggingface.co/ornith-ai/Ornith-1.5-9B-GGUF)

Pulled 2026-09-18:

| file | size | sha256 |
|---|---|---|
| `Ornith-1.5-9B-Q4_K_M.gguf` | 5,780,090,816 | `70c112196e0b7023803c9762752e46d29e612a92c83f995bc3ba1ceb07e8fab6` |

Vision mmproj was not pulled. MTP / `blk.32` nextn tensors are present in the GGUF and unused by this llama.cpp (`unused tensor blk.32.* -- ignoring`).

Same binary as Bonsai (Prism CUDA build, sm_75). Stock llama.cpp on this box is a SYCL build and was not used.

Serve command for the scored Ornith pass (Bonsai was stopped first; one process on :8097):

```bash
export LD_LIBRARY_PATH=/path/to/cudalibs:/path/to/build/bin
export CUDA_DEVICE_ORDER=PCI_BUS_ID
export CUDA_VISIBLE_DEVICES=1

./build/bin/llama-server \
  -m /path/to/Ornith-1.5-9B-Q4_K_M.gguf \
  -ngl 99 \
  -c 65536 \
  -fa on \
  --cache-type-k q4_0 \
  --cache-type-v q4_0 \
  --host 127.0.0.1 \
  --port 8097 \
  --jinja
```

Reported `/v1/models` meta: n_params 9197093888, n_ctx 65536, n_ctx_train 262144, ftype Q4_K Medium. VRAM while serving: about 6120 / 8192 MiB on the RTX 2080 SUPER. Process pid 762700. Client request body identical to the Bonsai scored pass. Do not add `chat_template_kwargs` mid-suite.
