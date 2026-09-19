
const $items = [{"json": {"feedUrl": "https://a.example/rss", "title": "new", "link": "https://a.example/1", "isoDate": "2023-11-14T12:00:00.000Z", "guid": "g1"}}, {"json": {"feedUrl": "https://a.example/rss", "title": "old", "link": "https://a.example/0", "isoDate": "2023-11-10T12:00:00.000Z", "guid": "g0"}}, {"json": {"feedUrl": "https://b.example/rss", "title": "dup-old", "link": "https://b.example/d", "isoDate": "2023-11-14T10:00:00.000Z", "guid": "dup"}}, {"json": {"feedUrl": "https://b.example/rss", "title": "dup-new", "link": "https://b.example/d2", "isoDate": "2023-11-14T11:00:00.000Z", "guid": "dup"}}, {"json": {"feedUrl": "not a url", "title": "badurl", "link": "https://x", "isoDate": "2023-11-14T12:00:00.000Z"}}, {"json": {"feedUrl": "https://c.example/rss", "title": "baddate", "link": "https://c", "isoDate": "nope"}}];
const $input = { all: () => [{"json": {"feedUrl": "https://a.example/rss", "title": "new", "link": "https://a.example/1", "isoDate": "2023-11-14T12:00:00.000Z", "guid": "g1"}}, {"json": {"feedUrl": "https://a.example/rss", "title": "old", "link": "https://a.example/0", "isoDate": "2023-11-10T12:00:00.000Z", "guid": "g0"}}, {"json": {"feedUrl": "https://b.example/rss", "title": "dup-old", "link": "https://b.example/d", "isoDate": "2023-11-14T10:00:00.000Z", "guid": "dup"}}, {"json": {"feedUrl": "https://b.example/rss", "title": "dup-new", "link": "https://b.example/d2", "isoDate": "2023-11-14T11:00:00.000Z", "guid": "dup"}}, {"json": {"feedUrl": "not a url", "title": "badurl", "link": "https://x", "isoDate": "2023-11-14T12:00:00.000Z"}}, {"json": {"feedUrl": "https://c.example/rss", "title": "baddate", "link": "https://c", "isoDate": "nope"}}] };
const Date_now_orig = Date.now;
Date.now = () => 1700000000000;
let RESULT;
try {
  RESULT = (function(){
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
  })();
} catch (e) {
  RESULT = { __threw: String(e && e.stack || e) };
}
Date.now = Date_now_orig;
console.log(JSON.stringify(RESULT));
