# Process notes

The scored Bonsai directory is a clean pass. Two earlier mistakes contaminated other directories; those files are kept so the record is inspectable rather than silently replaced.

## What is scored

`runs/bonsai/` was produced by one runner:

```
python3 run_suite.py --label bonsai --base-url http://127.0.0.1:8097/v1 \
    --model bonsai2-27b-ptq1_0 --out runs/bonsai --timeout 7200
```

Started 2026-09-18T17:07:47-0500, finished 2026-09-18T19:55:32-0500. Request body is the one in `harness/run_suite.py`. No `chat_template_kwargs` field. One process. The llama-server instance on `:8097` was not restarted mid-suite.

## What was discarded

`archive/bonsai-mixed-DISCARD/` is a Bonsai pass during which:

1. The request body was edited partway through the suite (an `enable_thinking` kwargs field was added). The rule for this eval was not to mix configurations; a rerun was required instead.
2. A second `run_suite.py` was launched against the same server and the same output directory. The two clients shared one 65,536-token KV pool. HTTP 500s followed.

A later probe on this endpoint suggested the extra kwargs field was a no-op (same seed, similar token counts). That does not make the mixed directory scorable. It is archive only.

T05 in that directory is an empty file (the second runner collided before it finished). T03 there is `finish_reason=length`, empty answer. T04 there contains SQL that fails on a schema-qualified `CREATE INDEX` name; useful as a qualitative note, not a score.

## Qwen3.8 traces

The vLLM build returns reasoning in `message.reasoning`. The first harness only read `reasoning_content`, so `archive/qwen38-notraces/` has answers without traces. The runner was patched to read either field. `runs/qwen38/` is the traced set except T03, which is a second sample: the first T03 returned empty `content` after 32,599 tokens and is preserved in the notraces archive.

`runs/qwen38/meta.json` was overwritten by that T03-only rerun. It has been reconstructed from the per-task markdown headers. The note is in the JSON.

## Client timeout vs model failure

An 1800 s urllib timeout used to cut Bonsai off while it was still generating. Those cuts are not model failures. The scored pass uses 7200 s. T04 and T05 on the scored pass still hit `max_tokens` with empty content; that is the model, not the client.

## Spark on the 2080 SUPER

Bonsai replaced a Spark-X2.5-4B server on this card for the eval. The restore command is:

```
./llama-server --model /home/patrick/models/Spark-X2.5-4B-Q4_K_M.gguf \
  --device Vulkan1 --split-mode none --n-gpu-layers 999 --ctx-size 491520 \
  --batch-size 128 --ubatch-size 128 --parallel 1 --flash-attn on \
  --cache-type-k q4_0 --cache-type-v q4_0 --fit off \
  --host 0.0.0.0 --port 8086 --jinja --threads 6
```

Paths are machine-local. Vulkan1 is the 2080 SUPER on Render-1's Vulkan build; the Bonsai server used CUDA, not Vulkan.

## Ornith-1.5-9B pass

`runs/ornith15-9b/` is a clean pass. One runner:

```
python3 run_suite.py --label ornith15-9b --base-url http://127.0.0.1:8097/v1 \
    --model /home/patrick/models/Ornith-1.5-9B-Q4_K_M.gguf \
    --out runs/ornith15-9b --timeout 7200
```

Started 2026-09-18T22:50:38-0500, finished 2026-09-18T23:49:40-0500. Bonsai llama-server was stopped first. Same request body as the scored Bonsai pass. No second runner. No mid-suite edit.

