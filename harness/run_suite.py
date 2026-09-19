#!/usr/bin/env python3
"""Run the coding suite against one OpenAI-compatible endpoint.

Both models are driven through /v1/chat/completions so the client code is
identical: Bonsai 2 27B via llama-server (PrismML fork) on Render-1, and
Qwen3.8-27B via vLLM on lawlab. Nothing here touches a real project; every
prompt and fixture lives in this sandbox directory.

Usage:
  python3 run_suite.py --label bonsai --base-url http://127.0.0.1:8090/v1 \
      --model bonsai2-27b --out runs/bonsai
  python3 run_suite.py --label qwen38 --base-url http://LAWLAB:8000/v1 \
      --model <served-name> --out runs/qwen38

Writes one .md per task plus meta.json (latency, token counts, params) so the
public repo carries the raw outputs, not just the scores.
"""
import argparse
import json
import pathlib
import time
import urllib.request

HERE = pathlib.Path(__file__).parent

SYSTEM = (
    "You are a senior software engineer. Answer the task exactly as asked. "
    "Do not add commentary the task did not request."
)


def load_tasks():
    tasks = []
    for line in (HERE / "prompts.jsonl").read_text().splitlines():
        if not line.strip():
            continue
        t = json.loads(line)
        if "fixture" in t:
            fx = (HERE / t["fixture"]).read_text()
            t["prompt"] = t["prompt_template"].replace("{FIXTURE}", fx)
        tasks.append(t)
    return tasks


def call(base_url, model, prompt, max_tokens, temperature, seed, timeout):
    body = {
        "model": model,
        "messages": [
            {"role": "system", "content": SYSTEM},
            {"role": "user", "content": prompt},
        ],
        "max_tokens": max_tokens,
        "temperature": temperature,
        "top_p": 0.95,
        "seed": seed,
        "stream": False,
    }
    req = urllib.request.Request(
        base_url.rstrip("/") + "/chat/completions",
        data=json.dumps(body).encode(),
        headers={"Content-Type": "application/json", "Authorization": "Bearer none"},
    )
    t0 = time.time()
    with urllib.request.urlopen(req, timeout=timeout) as r:
        data = json.load(r)
    elapsed = time.time() - t0
    msg = data["choices"][0]["message"]
    return {
        "content": msg.get("content") or "",
        "reasoning": msg.get("reasoning_content") or msg.get("reasoning") or "",
        "usage": data.get("usage", {}),
        "elapsed_s": round(elapsed, 2),
        "finish_reason": data["choices"][0].get("finish_reason"),
    }


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--label", required=True)
    ap.add_argument("--base-url", required=True)
    ap.add_argument("--model", required=True)
    ap.add_argument("--out", required=True)
    ap.add_argument("--temperature", type=float, default=0.2)
    ap.add_argument("--seed", type=int, default=1729)
    ap.add_argument("--timeout", type=int, default=1800)
    ap.add_argument("--only", default="", help="comma-separated task ids")
    args = ap.parse_args()

    out = pathlib.Path(args.out)
    out.mkdir(parents=True, exist_ok=True)
    only = {s.strip() for s in args.only.split(",") if s.strip()}

    meta = {
        "label": args.label,
        "model": args.model,
        "base_url": args.base_url,
        "temperature": args.temperature,
        "seed": args.seed,
        "started": time.strftime("%Y-%m-%dT%H:%M:%S%z"),
        "tasks": {},
    }

    for t in load_tasks():
        if only and t["id"] not in only:
            continue
        print(f"[{args.label}] {t['id']} {t['name']} ...", flush=True)
        try:
            r = call(args.base_url, args.model, t["prompt"], t["max_tokens"],
                     args.temperature, args.seed, args.timeout)
        except Exception as e:  # noqa: BLE001 - failures are a result, record them
            r = {"content": "", "reasoning": "", "usage": {}, "elapsed_s": None,
                 "finish_reason": f"ERROR: {type(e).__name__}: {e}"}
        (out / f"{t['id']}-{t['name']}.md").write_text(
            f"# {t['id']} {t['name']} ({args.label})\n\n"
            f"finish_reason: {r['finish_reason']}  elapsed: {r['elapsed_s']}s  "
            f"usage: {json.dumps(r['usage'])}\n\n"
            f"## Answer\n\n{r['content']}\n\n"
            + (f"## Thinking\n\n{r['reasoning']}\n" if r["reasoning"] else "")
        )
        meta["tasks"][t["id"]] = {
            "name": t["name"], "category": t["category"],
            "elapsed_s": r["elapsed_s"], "usage": r["usage"],
            "finish_reason": r["finish_reason"],
        }
        print(f"   {r['elapsed_s']}s  {r['usage']}", flush=True)

    meta["finished"] = time.strftime("%Y-%m-%dT%H:%M:%S%z")
    (out / "meta.json").write_text(json.dumps(meta, indent=2))
    print(f"wrote {out}")


if __name__ == "__main__":
    main()
