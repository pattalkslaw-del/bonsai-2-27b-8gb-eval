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