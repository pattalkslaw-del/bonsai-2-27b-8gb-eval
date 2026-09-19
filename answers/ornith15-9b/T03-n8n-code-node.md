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
