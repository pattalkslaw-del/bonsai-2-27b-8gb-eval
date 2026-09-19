
const $items = [{"json": {"feedUrl": "https://a.example/rss", "title": "new", "link": "https://a.example/1", "isoDate": "2023-11-14T12:00:00.000Z", "guid": "g1"}}, {"json": {"feedUrl": "https://a.example/rss", "title": "old", "link": "https://a.example/0", "isoDate": "2023-11-10T12:00:00.000Z", "guid": "g0"}}, {"json": {"feedUrl": "https://b.example/rss", "title": "dup-old", "link": "https://b.example/d", "isoDate": "2023-11-14T10:00:00.000Z", "guid": "dup"}}, {"json": {"feedUrl": "https://b.example/rss", "title": "dup-new", "link": "https://b.example/d2", "isoDate": "2023-11-14T11:00:00.000Z", "guid": "dup"}}, {"json": {"feedUrl": "not a url", "title": "badurl", "link": "https://x", "isoDate": "2023-11-14T12:00:00.000Z"}}, {"json": {"feedUrl": "https://c.example/rss", "title": "baddate", "link": "https://c", "isoDate": "nope"}}];
const $input = { all: () => [{"json": {"feedUrl": "https://a.example/rss", "title": "new", "link": "https://a.example/1", "isoDate": "2023-11-14T12:00:00.000Z", "guid": "g1"}}, {"json": {"feedUrl": "https://a.example/rss", "title": "old", "link": "https://a.example/0", "isoDate": "2023-11-10T12:00:00.000Z", "guid": "g0"}}, {"json": {"feedUrl": "https://b.example/rss", "title": "dup-old", "link": "https://b.example/d", "isoDate": "2023-11-14T10:00:00.000Z", "guid": "dup"}}, {"json": {"feedUrl": "https://b.example/rss", "title": "dup-new", "link": "https://b.example/d2", "isoDate": "2023-11-14T11:00:00.000Z", "guid": "dup"}}, {"json": {"feedUrl": "not a url", "title": "badurl", "link": "https://x", "isoDate": "2023-11-14T12:00:00.000Z"}}, {"json": {"feedUrl": "https://c.example/rss", "title": "baddate", "link": "https://c", "isoDate": "nope"}}] };
const Date_now_orig = Date.now;
Date.now = () => 1700000000000;
let RESULT;
try {
  RESULT = (function(){
    const items = $input.all() || [];

const now = Date.now();
const maxAgeMs = 48 * 60 * 60 * 1000;

let skipped = 0;
let dropped = 0;
let candidates = 0;

const byKey = new Map();

const getHostname = (url) => {
  if (typeof URL === 'function') {
    return new URL(url).hostname;
  }

  const match = /^https?:\/\/(?:[^@\/?#]*@)?([^\/?#]+)/i.exec(url);
  if (!match) {
    throw new Error('Invalid URL');
  }

  const host = match[1];

  if (host.startsWith('[')) {
    const end = host.indexOf(']');
    if (end === -1) {
      throw new Error('Invalid URL');
    }
    return host.slice(1, end);
  }

  const colon = host.indexOf(':');
  if (colon !== -1) {
    return host.slice(0, colon);
  }

  return host;
};

for (const item of items) {
  const json = item && item.json;

  if (!json || typeof json !== 'object') {
    skipped++;
    continue;
  }

  const isoDate = json.isoDate;

  if (typeof isoDate !== 'string' || isoDate.trim() === '') {
    skipped++;
    continue;
  }

  const timestamp = Date.parse(isoDate);

  if (!Number.isFinite(timestamp)) {
    skipped++;
    continue;
  }

  let publishedAt;

  try {
    publishedAt = new Date(timestamp).toISOString();
  } catch (e) {
    skipped++;
    continue;
  }

  const feedUrl = json.feedUrl;

  if (typeof feedUrl !== 'string' || feedUrl.trim() === '') {
    skipped++;
    continue;
  }

  let hostname;

  try {
    hostname = getHostname(feedUrl);
  } catch (e) {
    skipped++;
    continue;
  }

  if (!hostname) {
    skipped++;
    continue;
  }

  if (timestamp < now - maxAgeMs) {
    dropped++;
    continue;
  }

  const guid = json.guid;
  const link = json.link;

  const key =
    typeof guid === 'string' && guid.trim() !== ''
      ? guid
      : typeof link === 'string'
        ? link
        : undefined;

  if (key === undefined) {
    skipped++;
    continue;
  }

  candidates++;

  const existing = byKey.get(key);

  if (!existing || timestamp > existing.timestamp) {
    byKey.set(key, {
      title: json.title,
      link,
      timestamp,
      publishedAt,
      hostname,
    });
  }
}

dropped += Math.max(0, candidates - byKey.size);

const survivors = Array.from(byKey.values()).sort(
  (a, b) => b.timestamp - a.timestamp
);

const output = survivors.map(({ title, link, publishedAt, hostname }) => ({
  json: {
    title,
    link,
    publishedAt,
    sourceHost: hostname,
  },
}));

output.push({
  json: {
    summary: true,
    kept: survivors.length,
    dropped,
    skipped,
  },
});

return output;
  })();
} catch (e) {
  RESULT = { __threw: String(e && e.stack || e) };
}
Date.now = Date_now_orig;
console.log(JSON.stringify(RESULT));
