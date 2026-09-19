#!/usr/bin/env python3
"""Mechanical checks for the coding suite. Writes checks/results.json."""
import json, os, pathlib, re, shutil, subprocess, tempfile, textwrap

HERE = pathlib.Path(__file__).parent
RUNS = HERE / "runs"
OUT = HERE / "checks"
OUT.mkdir(exist_ok=True)


def answer(lab, tid):
    p = next((RUNS / lab).glob(f"{tid}-*.md"))
    t = p.read_text(errors="replace")
    if "## Answer" not in t:
        return ""
    return t.split("## Answer", 1)[1].split("## Thinking", 1)[0].strip()


def fences(s, lang=None):
    blocks = []
    for m in re.finditer(r"```(\w+)?\n(.*?)```", s, re.S):
        k, v = (m.group(1) or ""), m.group(2)
        if lang is None or k.lower() in lang:
            blocks.append(v)
    return blocks


def run(cmd, cwd=None, timeout=60, env=None):
    try:
        p = subprocess.run(
            cmd, cwd=cwd, timeout=timeout, env=env,
            stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True,
        )
        return p.returncode, p.stdout[-4000:]
    except Exception as e:  # noqa: BLE001
        return 99, f"{type(e).__name__}: {e}"


def strip_md(s):
    s = s.strip()
    if s.startswith("```"):
        s = re.sub(r"^```\w*\n", "", s)
        s = re.sub(r"\n```$", "", s)
    return s


def t01(lab):
    src = strip_md(answer(lab, "T01"))
    d = OUT / "t01-clean" / lab
    d.mkdir(parents=True, exist_ok=True)
    (d / "main.go").write_text(src + "\n")
    (d / "go.mod").write_text("module t01\n\ngo 1.22\n")
    # stub compile: package main already present
    rc, out = run(["gofmt", "-e", "main.go"], cwd=d)
    fmt_ok = rc == 0
    rc2, out2 = run(["go", "vet", "."], cwd=d)
    rc3, out3 = run(["go", "build", "-o", "t01.bin", "."], cwd=d)
    return {
        "gofmt_ok": fmt_ok, "gofmt": out[-500:],
        "govet_rc": rc2, "govet": out2[-800:],
        "build_rc": rc3, "build": out3[-800:],
        "compile_ok": rc3 == 0,
        "ident_bug_net_mail": "net/mail.ParseAddress" in src,
        "parseform_two_return": "form, err := r.ParseForm()" in src,
    }


def t02(lab):
    a = answer(lab, "T02")
    blocks = fences(a, {"ts", "typescript"})
    src = blocks[-1] if blocks else ""
    d = OUT / "t02" / lab
    d.mkdir(parents=True, exist_ok=True)
    (d / "route.ts").write_text(src)
    defects = []
    for needle, name in [
        ("await", "awaits_query"),
        ("return NextResponse.json({ error: 'not found'", "returns_404"),
        ("if (!id)", "null_id_check"),
        ("force-dynamic", "opts_out_next_cache"),
        ("cache", "keeps_module_cache"),
    ]:
        defects.append({"name": name, "present": needle in src})
    return {"src_chars": len(src), "checks": defects, "has_try": "try" in src}


N8N_HARNESS = r'''
const $items = INPUT;
const $input = { all: () => INPUT };
const Date_now_orig = Date.now;
Date.now = () => FROZEN;
let RESULT;
try {
  RESULT = (function(){
    BODY
  })();
} catch (e) {
  RESULT = { __threw: String(e && e.stack || e) };
}
Date.now = Date_now_orig;
console.log(JSON.stringify(RESULT));
'''


def t03(lab):
    src = strip_md(answer(lab, "T03"))
    d = OUT / "t03" / lab
    d.mkdir(parents=True, exist_ok=True)
    (d / "node.js").write_text(src)
    rc_syn, out_syn = run(["node", "--check", "node.js"], cwd=d)
    frozen = 1_700_000_000_000  # 2023-11-14
    items = [
        {"json": {"feedUrl": "https://a.example/rss", "title": "new", "link": "https://a.example/1",
                  "isoDate": "2023-11-14T12:00:00.000Z", "guid": "g1"}},
        {"json": {"feedUrl": "https://a.example/rss", "title": "old", "link": "https://a.example/0",
                  "isoDate": "2023-11-10T12:00:00.000Z", "guid": "g0"}},
        {"json": {"feedUrl": "https://b.example/rss", "title": "dup-old", "link": "https://b.example/d",
                  "isoDate": "2023-11-14T10:00:00.000Z", "guid": "dup"}},
        {"json": {"feedUrl": "https://b.example/rss", "title": "dup-new", "link": "https://b.example/d2",
                  "isoDate": "2023-11-14T11:00:00.000Z", "guid": "dup"}},
        {"json": {"feedUrl": "not a url", "title": "badurl", "link": "https://x", "isoDate": "2023-11-14T12:00:00.000Z"}},
        {"json": {"feedUrl": "https://c.example/rss", "title": "baddate", "link": "https://c", "isoDate": "nope"}},
    ]
    harness = (N8N_HARNESS
               .replace("INPUT", json.dumps(items))
               .replace("FROZEN", str(frozen))
               .replace("BODY", src))
    (d / "harness.js").write_text(harness)
    rc, out = run(["node", "harness.js"], cwd=d)
    parsed = None
    try:
        parsed = json.loads(out.strip().splitlines()[-1])
    except Exception:
        parsed = {"parse_error": out[-1000:]}
    uses_input_all = "$input.all" in src
    uses_items = "$items" in src
    return {
        "syntax_ok": rc_syn == 0, "syntax": out_syn[-400:],
        "run_rc": rc, "result": parsed,
        "uses_$input.all": uses_input_all, "uses_$items": uses_items,
    }


def t06(lab):
    a = answer(lab, "T06")
    blocks = fences(a, {"go"})
    # last two substantial go blocks: impl then tests, or scan
    impl = tests = None
    for b in blocks:
        if "func Fee(" in b and "testing" not in b:
            impl = b
        if "func TestFee" in b:
            tests = b
    d = OUT / "t06" / lab
    if d.exists():
        shutil.rmtree(d)
    d.mkdir(parents=True)
    (d / "go.mod").write_text("module t06\n\ngo 1.22\n")
    if not impl or not tests:
        return {"ok": False, "error": "missing go blocks", "nblocks": len(blocks)}
    # package name from impl
    pkg = "fee"
    m = re.search(r"^package (\w+)", impl, re.M)
    if m:
        pkg = m.group(1)
    (d / "fee.go").write_text(impl)
    (d / "fee_test.go").write_text(tests)
    rc, out = run(["go", "test", "."], cwd=d, timeout=90)
    return {"ok": rc == 0, "rc": rc, "out": out[-2500:], "pkg": pkg,
            "impl_chars": len(impl), "test_chars": len(tests)}


def t07(lab):
    a = answer(lab, "T07").lower()
    q1 = ("cachewarm" in a.replace(" ", "") or "cache_warm" in a or "cachewarm" in a.replace(" ", ""))
    q1b = "readindex" in a.replace(" ", "") or "errnotfound" in a.replace(" ", "") or "not found" in a
    q2 = "3" in a and "drainqueue" in a.replace(" ", "")
    q2b = "newclient" in a.replace(" ", "") or "defaultretrybudget" in a.replace(" ", "")
    q3 = "lockall" in a.replace(" ", "") and "flushsegment" in a.replace(" ", "")
    return {
        "q1_fn": "cachewarm" in a.replace(" ", ""),
        "q1_downstream": q1b,
        "q2_budget_3": "3" in a,
        "q2_set_site": q2b,
        "q2_exceed_fn": "drainqueue" in a.replace(" ", ""),
        "q3_pair": q3,
        "all_three": ("cachewarm" in a.replace(" ", "")) and q2 and q3,
        "answer": answer(lab, "T07"),
    }


TOOLS = {
    "list_matters": {"required": ["status"], "props": {"status", "limit"}},
    "get_invoice": {"required": ["matter_id"], "props": {"matter_id"}},
    "send_email": {"required": ["to", "subject", "body"], "props": {"to", "subject", "body"}},
}


def t08(lab):
    a = answer(lab, "T08").strip()
    # isolate JSON array
    i, j = a.find("["), a.rfind("]")
    blob = a[i:j+1] if i >= 0 and j > i else a
    try:
        calls = json.loads(blob)
        parse_ok = True
        err = None
    except Exception as e:
        calls, parse_ok, err = None, False, str(e)
    flags = []
    if parse_ok:
        names = [c.get("name") for c in calls]
        for c in calls:
            n = c.get("name")
            if n not in TOOLS:
                flags.append(f"invented_tool:{n}")
                continue
            args = c.get("arguments") or {}
            for r in TOOLS[n]["required"]:
                if r not in args:
                    flags.append(f"missing_required:{n}.{r}")
            for k in args:
                if k not in TOOLS[n]["props"]:
                    flags.append(f"unknown_arg:{n}.{k}")
        get_calls = [c for c in calls if c.get("name") == "get_invoice"]
        if len(get_calls) == 1:
            mid = (get_calls[0].get("arguments") or {}).get("matter_id", "")
            if "[*]" not in str(mid) and "matters" in str(mid):
                flags.append("get_invoice_whole_list_not_per_id")
            flags.append("single_get_invoice_not_per_matter")
        if "list_matters" not in names:
            flags.append("no_list_matters")
        if "send_email" not in names:
            flags.append("no_send_email")
        else:
            em = next(c for c in calls if c["name"] == "send_email")
            if (em.get("arguments") or {}).get("to") != "billing@example.com":
                flags.append("wrong_email_to")
    return {"parse_ok": parse_ok, "error": err, "n_calls": len(calls) if calls else 0,
            "flags": flags, "calls": calls}


def t04_note(lab):
    a = answer(lab, "T04")
    return {"empty": not a, "chars": len(a),
            "schema_qualified_index": "CREATE INDEX IF NOT EXISTS public." in a,
            "has_begin": a.strip().startswith("BEGIN") or "BEGIN;" in a[:40]}


def t05_note(lab):
    a = answer(lab, "T05")
    low = a.lower()
    return {
        "empty": not a,
        "chars": len(a),
        "mentions_temp_server": "temporary" in low,
        "mentions_initdb": "init" in low and ("postgres" in low or "database" in low),
        "mentions_pg_isready": "pg_isready" in low,
        "mentions_healthcheck": "health" in low,
    }


def main():
    results = {}
    for lab in ["bonsai", "qwen38"]:
        results[lab] = {
            "T01": t01(lab),
            "T02": t02(lab),
            "T03": t03(lab),
            "T04": t04_note(lab),
            "T05": t05_note(lab),
            "T06": t06(lab),
            "T07": t07(lab),
            "T08": t08(lab),
        }
    (OUT / "results.json").write_text(json.dumps(results, indent=2, default=str))
    print("wrote", OUT / "results.json")
    for lab, tasks in results.items():
        print("==", lab)
        for tid, r in tasks.items():
            print(" ", tid, {k: v for k, v in r.items() if k not in ("answer", "calls", "result") and not str(k).endswith("chars")})


if __name__ == "__main__":
    main()
