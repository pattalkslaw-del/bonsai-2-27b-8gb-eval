# T03 n8n-code-node (ornith15-9b)

finish_reason: stop  elapsed: 107.99s  usage: {"completion_tokens": 7364, "prompt_tokens": 248, "total_tokens": 7612, "prompt_tokens_details": {"cached_tokens": 29}}

## Answer

```javascript
import { INode, INodeExecutionData } from 'n8n-workflow';

export const nodes: INode = {
    name: 'Code',
    type: 'n8n-nodes-base.code',
    typeVersion: 2,
    position: [250, 300],
    properties: {
        [`${'$input'}`]: {
            value: {
                first: [],
                all: [],
            },
        },
    },
};

const FORTY_EIGHT_HOURS_MS = 48 * 60 * 60 * 1000;

export const output = (items: INodeExecutionData[]): INodeExecutionData[] => {
    const now = Date.now();
    let dropped = 0;
    let skipped = 0;

    // Phase 1 — validate, then drop stale items
    const candidates: INodeExecutionData[] = [];

    for (const item of items) {
        const json = item.json;

        let isoDate: Date;
        try {
            isoDate = new Date(json.isoDate);
            if (Number.isNaN(isoDate.getTime())) throw new Error('invalid isoDate');
        } catch {
            skipped++;
            continue;
        }

        let sourceHost: string;
        try {
            sourceHost = new URL(json.feedUrl).hostname;
        } catch {
            skipped++;
            continue;
        }

        if (now - isoDate.getTime() > FORTY_EIGHT_HOURS_MS) {
            dropped++;
            continue;
        }

        candidates.push({
            ...item,
            json: { ...json, isoDate, sourceHost },
        });
    }

    // Phase 2 — deduplicate, keeping the newest per key (guid ?? link)
    const byKey = new Map<string, INodeExecutionData>();

    for (const candidate of candidates) {
        const key = candidate.json.guid ?? candidate.json.link;
        const existing = byKey.get(key);

        if (
            !existing ||
            candidate.json.isoDate.getTime() > existing.json.isoDate.getTime()
        ) {
            byKey.set(key, candidate);
        }
    }

    // Phase 3 — sort survivors newest first
    const survivors = Array.from(byKey.values()).sort(
        (a, b) => b.json.isoDate.getTime() - a.json.isoDate.getTime(),
    );

    // Phase 4 — emit survivors
    const result: INodeExecutionData[] = survivors.map((s) => {
        const json = s.json;
        return {
            json: {
                title: json.title,
                link: json.link,
                publishedAt: json.isoDate.toISOString(),
                sourceHost: new URL(json.feedUrl).hostname,
            },
        };
    });

    // Phase 5 — summary
    result.push({
        json: {
            summary: true,
            kept: survivors.length,
            dropped,
            skipped,
        },
    });

    return result;
};
```

## Thinking

The user wants me to write the JavaScript body of an n8n Code node. Let me think about the structure of an n8n Code node.

An n8n Code node has this structure:

```javascript
import { INodeParameters, INodeExecutionData } from 'n8n-workflow';

export const nodes: INode = {
    name: 'Code',
    type: 'n8n-nodes-base.code',
    typeVersion: 2,
    position: [250, 300],
    credentials: {
        // ...
    },
    properties: {
        resource: { value: 'execution' },
        operation: { value: 'start' },
        [`${'$input'}`]: {
            value: {
                first: [],
                all: [],
            },
        },
    },
};

export const input = (item: INodeExecutionData): INodeExecutionData => {
    // ...
};

export const output = (items: INodeExecutionData[]): INodeExecutionData[] => {
    // ...
};
```

Let me write the logic:

1. For each input item, parse `isoDate` to a Date. If malformed, skip and count.
2. Parse `feedUrl` to get hostname. If malformed, skip and count.
3. Drop items older than 48 hours relative to now.
4. Deduplicate on `guid` when present, otherwise on `link`, keeping the newest by `isoDate`.
5. Sort survivors newest first.
6. Emit each survivor as `{ json: { title, link, publishedAt, sourceHost } }`.
7. Emit final item `{ json: { summary: true, kept, dropped, skipped } }`.

Let me think about the details.

For the 48-hour drop: "Drop items older than 48 hours relative to now." So if `now - isoDate > 48 hours`, drop it.

For deduplication: "Deduplicate on `guid` when present, otherwise on `link`, keeping the newest by `isoDate`." So for each item, the key is `guid` if present, else `link`. Keep the newest (largest isoDate).

Wait, but what if some items have guid and some don't? The dedup key is per-item: if an item has a guid, use guid as key; otherwise use link. So items are grouped by their key.

Let me think about the order of operations. Should I drop old items first, then dedupe? Or dedupe first, then drop? The spec lists them in order: drop first, then dedupe. Let me follow that order.

Actually, let me reconsider. The order matters for the counts. Let me follow the spec order:
1. Drop items older than 48 hours.
2. Deduplicate.
3. Sort.
4. Emit.

Let me think about what counts mean:
- `dropped`: items dropped because older than 48 hours.
- `skipped`: items skipped because malformed isoDate or feedUrl.
- `kept`: survivors after dedup.

Now, should skipped items be counted before or after the 48-hour drop? The spec says "Never throw on a malformed `isoDate` or `feedUrl`; skip that item instead and count it." I think skipped should be counted regardless. Let me think about the order.

Actually, if an item has a malformed isoDate, we can't determine its age, so we skip it. If it has a malformed feedUrl, we can't get the hostname, so we skip it. These should be counted as skipped.

Let me think about whether to check malformed first or age first. If isoDate is malformed, we can't check age, so we must skip. So the logic per item:
- Try to parse isoDate. If fails, skip (count skipped).
- Try to parse feedUrl hostname. If fails, skip (count skipped).
- Check age. If older than 48 hours, drop (count dropped).
- Otherwise, it's a candidate for dedup.

Now for the hostname parsing: `feedUrl` is a string. We need the hostname. We can use `new URL(feedUrl).hostname`. If it throws, skip.

Let me write the code.

```javascript
import { INodeExecutionData } from 'n8n-workflow';

const FORTY_EIGHT_HOURS = 48 * 60 * 60 * 1000;

export const input = (item: INodeExecutionData): INodeExecutionData => {
    return item;
};

export const output = (items: INodeExecutionData[]): INodeExecutionData[] => {
    const now = Date.now();
    const kept: INodeExecutionData[] = [];
    let dropped = 0;
    let skipped = 0;

    // Phase 1: filter by age and validity
    const candidates: INodeExecutionData[] = [];

    for (const item of items) {
        const json = item.json;

        // Parse isoDate
        let isoDate: Date;
        try {
            isoDate = new Date(json.isoDate);
            if (Number.isNaN(isoDate.getTime())) {
                throw new Error('Invalid date');
            }
        } catch {
            skipped++;
            continue;
        }

        // Parse feedUrl hostname
        let sourceHost: string;
        try {
            sourceHost = new URL(json.feedUrl).hostname;
        } catch {
            skipped++;
            continue;
        }

        // Age check
        if (now - isoDate.getTime() > FORTY_EIGHT_HOURS) {
            dropped++;
            continue;
        }

        candidates.push({
            ...item,
            json: {
                ...json,
                isoDate,
                sourceHost,
            },
        });
    }

    // Phase 2: deduplicate
    const dedupeMap = new Map<string, INodeExecutionData>();

    for (const candidate of candidates) {
        const json = candidate.json;
        const key = json.guid ?? json.link;

        const existing = dedupeMap.get(key);
        if (!existing) {
            dedupeMap.set(key, candidate);
            continue;
        }

        // Keep the newest by isoDate
        if (candidate.json.isoDate.getTime() > existing.json.isoDate.getTime()) {
            dedupeMap.set(key, candidate);
        }
    }

    const survivors = Array.from(dedupeMap.values());

    // Phase 3: sort newest first
    survivors.sort((a, b) => b.json.isoDate.getTime() - a.json.isoDate.getTime());

    // Phase 4: emit
    const outputItems: INodeExecutionData[] = survivors.map((s) => {
        const json = s.json;
        return {
            json: {
                title: json.title,
                link: json.link,
                publishedAt: json.isoDate.toISOString(),
                sourceHost,
            },
        };
    });

    outputItems.push({
        json: {
            summary: true,
            kept: survivors.length,
            dropped,
            skipped,
        },
    });

    return outputItems;
};
```

Wait, I have a bug. In the `survivors.map`, I reference `sourceHost` but it's scoped inside the loop. Let me fix that. I need to capture it per survivor.

Let me restructure to capture sourceHost per survivor.

```javascript
    // Phase 4: emit
    const outputItems: INodeExecutionData[] = survivors.map((s) => {
        const json = s.json;
        const sourceHost = new URL(json.feedUrl).hostname;
        return {
            json: {
                title: json.title,
                link: json.link,
                publishedAt: json.isoDate.toISOString(),
                sourceHost,
            },
        };
    });
```

That's cleaner. But wait, we already validated feedUrl in phase 1, so it should be fine.

Actually, let me reconsider the structure. The `input` function in n8n Code nodes is optional. In newer versions, you can just use `output`. But the standard structure includes both. Let me include `input` as a passthrough for compatibility, or just omit it. Actually, in n8n, the Code node's `output` receives the array of items. The `input` function is used to process individual items before they're passed to output. If you don't need it, you can omit it.

Let me think about the n8n Code node type version. In n8n, the Code node has typeVersion 2 (older) or 3 (newer). Let me use a reasonable structure.

Actually, the modern n8n Code node (typeVersion 3) uses a different structure with `execute` function. Let me think about which to use.

The classic structure (typeVersion 2) uses `input` and `output` functions. This is widely used and well-documented.

The newer structure (typeVersion 3) uses:

```javascript
import { INode, INodeProperties, INodeExecutionData } from 'n8n-workflow';

export const nodes: INode = {
    name: 'Code',
    type: 'n8n-nodes-base.code',
    typeVersion: 3,
    position: [250, 300],
    properties: {
        [`${'$input'}`]: {
            value: {
                first: [],
                all: [],
            },
        },
    },
};

export const execute = (
    items: INodeExecutionData[],
): INodeExecutionData[] => {
    // ...
};
```

Hmm, but the question asks for "the JavaScript body of an n8n Code node". I think they want the code that goes in the code editor, which is the `input`/`output` or `execute` function. Let me provide the full node definition to be safe, but focus on the logic.

Actually, "Write the JavaScript body of an n8n Code node" - I think they want the code that goes in the code editor. Let me provide the full node definition to be safe, but focus on the logic.

Let me reconsider. The most common and portable form is the `input`/`output` structure. Let me use that. I'll include the node definition and the functions.

Let me reconsider the counts semantics once more. The spec says:
- "Drop items older than 48 hours relative to now." → dropped count
- "Deduplicate..." → survivors
- "Emit each survivor" → kept
- "Never throw on a malformed `isoDate` or `feedUrl`; skip that item instead and count it." → skipped count
- "Emit one final item `{ json: { summary: true, kept, dropped, skipped } }`."

So the counts are: kept = number of survivors, dropped = number dropped by age, skipped = number skipped by malformed data.

Now, one question: should the summary item itself be counted in `kept`? No, `kept` is the number of survivors (articles). The summary is separate.

Let me also reconsider: should dropped items that are also malformed be counted as dropped or skipped? If isoDate is malformed, we skip (can't determine age). So malformed takes precedence. That's what I have.

Let me also think about edge cases:
- What if `guid` is present but empty string? `json.guid ?? json.link` - empty string is not null/undefined, so it would use empty string as key. That's probably fine, but maybe we should treat empty as absent? The spec says "when present". An empty string is technically present. Let me keep it simple and use `??`. Actually, to be safe, maybe I should check for presence more carefully. But the spec says "when present", so `guid` being present means it has a value. Let me use `json.guid ?? json.link`. Hmm, but if guid is an empty string, that's a weird edge case. Let me just follow the spec literally.

Actually, let me reconsider. The input type says `guid?: string`, meaning guid is optional. So when present, use guid; otherwise use link. `json.guid ?? json.link` handles this correctly for null/undefined.

Let me also think about the `publishedAt` format. "an ISO-8601 string in UTC". `Date.toISOString()` returns something like `2024-01-01T00:00:00.000Z`, which is ISO-8601 in UTC. Good.

Now, should I preserve the original isoDate or use the parsed one? I use the parsed one for publishedAt, which is correct since it's the same instant.

Let me also reconsider whether to store isoDate on the candidate. I add it to make dedup and sort easier. That's fine.

Let me finalize the code. I'll use the `input`/`output` structure.

Actually, let me reconsider the n8n Code node structure more carefully. In n8n, when you use a Code node, the code you write is wrapped. The `output` function receives the items array. Let me provide a clean, complete version.

Here's my final version:

```javascript
import { INodeExecutionData } from 'n8n-workflow';

const FORTY_EIGHT_HOURS_MS = 48 * 60 * 60 * 1000;

export const input = (item: INodeExecutionData): INodeExecutionData => {
    return item;
};

export const output = (items: INodeExecutionData[]): INodeExecutionData[] => {
    const now = Date.now();
    let dropped = 0;
    let skipped = 0;

    // Phase 1 — validate and drop stale items
    const candidates: INodeExecutionData[] = [];

    for (const item of items) {
        const json = item.json;

        let isoDate: Date;
        try {
            isoDate = new Date(json.isoDate);
            if (Number.isNaN(isoDate.getTime())) throw new Error('bad date');
        } catch {
            skipped++;
            continue;
        }

        let sourceHost: string;
        try {
            sourceHost = new URL(json.feedUrl).hostname;
        } catch {
            skipped++;
            continue;
        }

        if (now - isoDate.getTime() > FORTY_EIGHT_HOURS_MS) {
            dropped++;
            continue;
        }

        candidates.push({
            ...item,
            json: { ...json, isoDate, sourceHost },
        });
    }

    // Phase 2 — deduplicate, keeping the newest per key
    const byKey = new Map<string, INodeExecutionData>();

    for (const candidate of candidates) {
        const key = candidate.json.guid ?? candidate.json.link;
        const existing = byKey.get(key);

        if (!existing || candidate.json.isoDate.getTime() > existing.json.isoDate.getTime()) {
            byKey.set(key, candidate);
        }
    }

    const survivors = Array.from(byKey.values()).sort(
        (a, b) => b.json.isoDate.getTime() - a.json.isoDate.getTime(),
    );

    // Phase 3 — emit survivors
    const result: INodeExecutionData[] = survivors.map((s) => {
        const json = s.json;
        return {
            json: {
                title: json.title,
                link: json.link,
                publishedAt: json.isoDate.toISOString(),
                sourceHost: new URL(json.feedUrl).hostname,
            },
        };
    });

    // Phase 4 — summary
    result.push({
        json: { summary: true, kept: survivors.length, dropped, skipped },
    });

    return result;
};
```

This looks good. Let me double-check the logic once more.

1. Age check: `now - isoDate.getTime() > FORTY_EIGHT_HOURS_MS` → drop. Correct.
2. Dedup: key = guid ?? link. Keep newest. Correct.
3. Sort: newest first. Correct.
4. Emit: title, link, publishedAt (toISOString UTC), sourceHost (hostname). Correct.
5. Summary: kept, dropped, skipped. Correct.

One thing: the `input` function is a passthrough. In n8n, if you only define `output`, it still works. But defining `input` as passthrough is harmless. Actually, let me reconsider whether to include it. In n8n Code node typeVersion 2, both `input` and `output` are supported. The `input` function processes each item individually. If I don't define it, n8n will just pass items directly to `output`. So I can omit `input`. But including it as passthrough is fine and explicit. Let me keep it minimal and omit `input` since it's not needed. Actually, let me include it for clarity and compatibility. Hmm.

Actually, I'll omit the `input` function since it's not needed and keeps the code focused. Wait, but some n8n versions might require it? No, `output` alone is sufficient. Let me omit it.

Hmm, but actually, let me reconsider. The question says "Write the JavaScript body of an n8n Code node". This suggests they want the code that goes in the code editor. The code editor for a Code node contains the `input`/`output` or `execute` functions. Let me provide the complete node definition plus the functions, or just the functions?

I think providing the full node definition is most complete and unambiguous. But the "body" might just mean the functions. Let me provide the full structure to be safe, since it's a self-contained node.

Actually, let me reconsider. The phrase "JavaScript body" suggests the code inside the code editor. I'll provide the full node definition with the functions, as that's the complete, runnable code. But I could also just provide the functions. Let me provide the full node definition for completeness, since that's what you'd paste into the node.

Hmm, but the full node definition with `nodes` export is the file-level structure. The "body" of the code editor is the functions. Let me provide both the node definition and the functions to be complete.

Let me reconsider the type version. For n8n 1.x, the Code node typeVersion is 2 (with input/output) or 3 (with execute). Let me use typeVersion 2 with input/output, which is the most widely compatible.

Actually, I realize I should double check: does the Code node's `output` receive the array of all items? Yes. And `input` receives each item individually. So my structure is correct.

Let me also reconsider: should I handle the case where `items` is empty? If empty, candidates is empty, survivors is empty, result has just the summary with kept=0, dropped=0, skipped=0. That's correct.

Let me also reconsider the `publishedAt` — should it be the parsed isoDate converted to UTC ISO? Yes, `json.isoDate.toISOString()` gives UTC ISO-8601. Correct.

Now, one more consideration: the spec says "publishedAt is an ISO-8601 string in UTC". `toISOString()` always returns UTC with 'Z' suffix. Correct.

Let me also think about whether `sourceHost` should be lowercased. `URL.hostname` returns lowercase. That's fine.

I think the code is correct. Let me finalize.

Actually, let me reconsider whether to include the node definition. The question says "Return only the code." So I should return only the code, no commentary. Let me provide the full node definition with the functions.

Wait, "Return only the code" means don't add commentary. So I should just output the code without explanation. Let me do that.

Let me reconsider the structure once more. I'll provide the complete n8n Code node.

Actually, I want to reconsider whether the `input` function should be there. Let me think about n8n's Code node behavior:

- Type version 2: Has `input` and `output` functions. `input` is called for each item, `output` is called with the array of items.
- Type version 3: Has `execute` function called with the array of items.

For maximum compatibility and clarity, I'll use type version 2 with `input`/`output`. But since `input` isn't needed, I could omit it. Let me include it as a passthrough to be explicit and safe, since some setups might expect it. Actually no, omitting is cleaner. Let me omit it.

Hmm, let me reconsider. If I omit `input`, will n8n complain? No. The `output` function is sufficient. Let me omit `input`.

Wait, actually, I want to be careful. Let me reconsider whether the Code node requires the `input` function. In n8n, the Code node's `output` function receives the items. The `input` function is optional and used for per-item processing. If you only define `output`, it works fine. So I'll omit `input`.

Let me finalize with the full node definition and just the `output` function.

Actually, I realize I should double-check: in n8n, is it `output` that receives the array? Let me confirm. Yes, in the Code node, the `output` function receives the array of items and returns the array of output items. The `input` function receives each item individually.

OK here's my final answer. Let me write it cleanly.

Let me reconsider whether to use `input`/`output` or `execute`. I'll go with `input`/`output` (typeVersion 2) as it's the most common and well-understood.

Final code:

```javascript
import { INodeExecutionData } from 'n8n-workflow';

const FORTY_EIGHT_HOURS_MS = 48 * 60 * 60 * 1000;

export const nodes: INode = {
    name: 'Code',
    type: 'n8n-nodes-base.code',
    typeVersion: 2,
    position: [250, 300],
    properties: {
        [`${'$input'}`]: {
            value: {
                first: [],
                all: [],
            },
        },
    },
};

export const output = (items: INodeExecutionData[]): INodeExecutionData[] => {
    const now = Date.now();
    let dropped = 0;
    let skipped = 0;

    const candidates: INodeExecutionData[] = [];

    for (const item of items) {
        const json = item.json;

        let isoDate: Date;
        try {
            isoDate = new Date(json.isoDate);
            if (Number.isNaN(isoDate.getTime())) throw new Error('invalid isoDate');
        } catch {
            skipped++;
            continue;
        }

        let sourceHost: string;
        try {
            sourceHost = new URL(json.feedUrl).hostname;
        } catch {
            skipped++;
            continue;
        }

        if (now - isoDate.getTime() > FORTY_EIGHT_HOURS_MS) {
            dropped++;
            continue;
        }

        candidates.push({
            ...item,
            json: { ...json, isoDate, sourceHost },
        });
    }

    const byKey = new Map<string, INodeExecutionData>();

    for (const candidate of candidates) {
        const key = candidate.json.guid ?? candidate.json.link;
        const existing = byKey.get(key);

        if (
            !existing ||
            candidate.json.isoDate.getTime() > existing.json.isoDate.getTime()
        ) {
            byKey.set(key, candidate);
        }
    }

    const survivors = Array.from(byKey.values()).sort(
        (a, b) => b.json.isoDate.getTime() - a.json.isoDate.getTime(),
    );

    const result: INodeExecutionData[] = survivors.map((s) => {
        const json = s.json;
        return {
            json: {
                title: json.title,
                link: json.link,
                publishedAt: json.isoDate.toISOString(),
                sourceHost: new URL(json.feedUrl).hostname,
            },
        };
    });

    result.push({
        json: {
            summary: true,
            kept: survivors.length,
            dropped,
            skipped,
        },
    });

    return result;
};
```

I need to import `INode` too. Let me add that.

Actually, let me reconsider whether to include the node definition at all. The question asks for "the JavaScript body of an n8n Code node". I think the most useful answer is the code that goes in the code editor, which is the `output` function (and optionally `input`). But providing the full node definition makes it self-contained and clear.

Hmm, I'll provide the full node definition since it's complete and unambiguous. But I could also just provide the functions. Let me provide the full node definition for completeness.

Wait, actually, re-reading: "Write the JavaScript body of an n8n Code node running in 'Run Once for All Items' mode." The 'Run Once for All Items' mode is a setting of the Code node that determines whether `input`/`output` or `execute` is used. In 'Run Once for All Items', the `output` function receives all items at once. This confirms I should use the `output` function approach.

So the code editor body for 'Run Once for All Items' mode is the `output` function (and optionally `input`). Let me provide the full node definition plus the functions.

Let me finalize. I'll include the import for both types.

Actually, I realize I should double check the exact import names. In n8n-workflow:
- `INodeExecutionData` - yes
- `INode` - yes

Both are exported from 'n8n-workflow'. Good.

Let me write the final answer. I'll provide the complete node definition and the output function.

Actually, let me reconsider whether to include the `input` function. Since 'Run Once for All Items' mode uses `output` (not `input`/`output` per-item), and I don't need per-item processing, I'll omit `input`. The `output` function alone is sufficient.

Hmm, but wait. Let me reconsider. In 'Run Once for All Items' mode, does the Code node use `input`/`output` or `execute`? Let me think.

In n8n:
- 'Run Once for All Items' mode: The `output` function receives the array of all items.
- 'Run Once for Each Item' mode: The `input` function is called for each item.

So 'Run Once for All Items' mode uses `output`. Good. I'll provide the `output` function.

But actually, the type version also matters. Type version 2 uses `input`/`output`. Type version 3 uses `execute`. The mode setting ('Run Once for All Items' vs 'Run Once for Each Item') is a separate setting.

For 'Run Once for All Items' mode with type version 2, the `output` function receives all items. This is what I have. Good.

Let me finalize.

Actually, I want to make sure the code is robust. Let me reconsider the `new URL` parsing. If `feedUrl` is not a valid URL (e.g., "not a url"), `new URL` throws, and we skip. Good. If `feedUrl` is empty string, `new URL('')` throws, skip. Good.

Also, `new URL('http://example.com/path')` gives hostname 'example.com'. Good. `new URL('https://sub.example.com:8080/path')` gives hostname 'sub.example.com'. Good.

Let me also reconsider: what if `isoDate` is a valid date but in the future? `now - isoDate.getTime()` would be negative, which is not > 48 hours, so it's kept. That's correct behavior (future dates are not "older than 48 hours").

OK, I'm confident in the code. Let me write the final answer.

Let me reconsider one more time whether I should include the node definition or just the functions. The question says "Write the JavaScript body of an n8n Code node". I'll interpret "body" as the code in the editor. I'll provide the full node definition to be complete and self-contained, since that's what you'd paste into the node's code editor.

Hmm, actually, the code editor for an n8n Code node contains the functions (input/output or execute), not the node definition. The node definition is in the node's properties/config, not the code editor. So "the JavaScript body" likely refers to the functions in the code editor.

But to be safe and complete, I'll provide the full node definition plus the functions. This way, it's unambiguous and complete.

Actually, let me just provide the functions since that's what "body" most likely means, but I'll include the node definition too for completeness. Let me provide the full thing.

Let me write it out cleanly.

Final decision: Provide the complete node definition with the `output` function. This is self-contained and complete.

Let me make sure the code compiles mentally:
- Import `INodeExecutionData` and `INode` from 'n8n-workflow'.
- Define `nodes` object.
- Define `output` function.
- Inside `output`, use `Date.now()`, loops, Map, sort, map, push.
- Return result.

Looks good. Let me write it.

One more thing: I should make sure I don't add any commentary. The question says "Return only the code." So I'll just output the code.

Let me finalize.

