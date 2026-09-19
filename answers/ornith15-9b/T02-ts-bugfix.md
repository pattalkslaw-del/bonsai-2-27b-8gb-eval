# T02 ts-bugfix (ornith15-9b)

finish_reason: stop  elapsed: 57.13s  usage: {"completion_tokens": 3944, "prompt_tokens": 255, "total_tokens": 4199, "prompt_tokens_details": {"cached_tokens": 29}}


## Answer

## Defects

1. **Race condition (stale data):** The query result is assigned inside the `.then()` callback, but the handler synchronously checks `result` and sends the response before the query resolves, so `result` is always `undefined` at that point and the real data is never returned.

2. **Cannot set headers after they are sent:** `NextResponse.json()` is invoked twice (in the 404 branch and in the `return`), so when both execute, the second call throws because the response was already sent.

3. **Reference-based cache (stale data):** `cache[id!] = result` stores the result object by reference, so mutating a returned object corrupts the cached copy and makes subsequent cache hits return stale data.

4. **Missing null check on `id`:** `searchParams.get('id')` can return `null`, and `id` is used unguarded in `cache[id!]` and the SQL query, causing a null reference or invalid query.

## Corrected file

```ts
import { NextRequest, NextResponse } from 'next/server';
import { db } from '@/lib/db';

interface Matter {
  id: string | number;
  [key: string]: unknown;
}

const cache = new Map<string, Matter>();

export async function GET(req: NextRequest) {
  const id = req.nextUrl.searchParams.get('id');
  if (!id) {
    return NextResponse.json({ error: 'missing id' }, { status: 400 });
  }

  if (cache.has(id)) {
    return NextResponse.json(cache.get(id));
  }

  const result = await db.query('SELECT * FROM matters WHERE id = $1', [id]);
  const row = result.rows[0];

  if (!row) {
    return NextResponse.json({ error: 'not found' }, { status: 404 });
  }

  cache.set(id, { ...row });
  return NextResponse.json(row);
}
```
