
const $items = [{"json": {"feedUrl": "https://a.example/rss", "title": "new", "link": "https://a.example/1", "isoDate": "2023-11-14T12:00:00.000Z", "guid": "g1"}}, {"json": {"feedUrl": "https://a.example/rss", "title": "old", "link": "https://a.example/0", "isoDate": "2023-11-10T12:00:00.000Z", "guid": "g0"}}, {"json": {"feedUrl": "https://b.example/rss", "title": "dup-old", "link": "https://b.example/d", "isoDate": "2023-11-14T10:00:00.000Z", "guid": "dup"}}, {"json": {"feedUrl": "https://b.example/rss", "title": "dup-new", "link": "https://b.example/d2", "isoDate": "2023-11-14T11:00:00.000Z", "guid": "dup"}}, {"json": {"feedUrl": "not a url", "title": "badurl", "link": "https://x", "isoDate": "2023-11-14T12:00:00.000Z"}}, {"json": {"feedUrl": "https://c.example/rss", "title": "baddate", "link": "https://c", "isoDate": "nope"}}];
const $input = { all: () => [{"json": {"feedUrl": "https://a.example/rss", "title": "new", "link": "https://a.example/1", "isoDate": "2023-11-14T12:00:00.000Z", "guid": "g1"}}, {"json": {"feedUrl": "https://a.example/rss", "title": "old", "link": "https://a.example/0", "isoDate": "2023-11-10T12:00:00.000Z", "guid": "g0"}}, {"json": {"feedUrl": "https://b.example/rss", "title": "dup-old", "link": "https://b.example/d", "isoDate": "2023-11-14T10:00:00.000Z", "guid": "dup"}}, {"json": {"feedUrl": "https://b.example/rss", "title": "dup-new", "link": "https://b.example/d2", "isoDate": "2023-11-14T11:00:00.000Z", "guid": "dup"}}, {"json": {"feedUrl": "not a url", "title": "badurl", "link": "https://x", "isoDate": "2023-11-14T12:00:00.000Z"}}, {"json": {"feedUrl": "https://c.example/rss", "title": "baddate", "link": "https://c", "isoDate": "nope"}}] };
const Date_now_orig = Date.now;
Date.now = () => 1700000000000;
let RESULT;
try {
  RESULT = (function(){
    const toSafeString = (value) => {
  try {
    return value == null ? '' : String(value);
  } catch (e) {
    return '';
  }
};

const items = Array.isArray($items) ? $items : [];
const now = Date.now();
const cutoff = now - 48 * 60 * 60 * 1000;

let dropped = 0;
let skipped = 0;
const best = new Map();

for (const item of items) {
  const json = item?.json || {};

  let parsed;
  try {
    parsed = new Date(json.isoDate);
  } catch (e) {
    parsed = null;
  }

  if (parsed === null || Number.isNaN(parsed.getTime())) {
    skipped += 1;
    continue;
  }

  let sourceHost;
  try {
    const url = new URL(toSafeString(json.feedUrl));
    sourceHost = url.hostname;
  } catch (e) {
    skipped += 1;
    continue;
  }

  if (!sourceHost) {
    skipped += 1;
    continue;
  }

  if (parsed.getTime() < cutoff) {
    dropped += 1;
    continue;
  }

  const guid = toSafeString(json.guid).trim();
  const link = toSafeString(json.link);
  const title = toSafeString(json.title);
  const key = guid !== '' ? guid : link;

  const existing = best.get(key);
  if (!existing || parsed.getTime() > existing.date.getTime()) {
    best.set(key, {
      date: parsed,
      title,
      link,
      sourceHost,
    });
  }
}

const survivors = Array.from(best.values()).sort((a, b) => b.date.getTime() - a.date.getTime());
const kept = survivors.length;

return [
  ...survivors.map(({ title, link, sourceHost, date }) => ({
    json: {
      title,
      link,
      publishedAt: date.toISOString(),
      sourceHost,
    },
  })),
  {
    json: {
      summary: true,
      kept,
      dropped,
      skipped,
    },
  },
];
  })();
} catch (e) {
  RESULT = { __threw: String(e && e.stack || e) };
}
Date.now = Date_now_orig;
console.log(JSON.stringify(RESULT));
