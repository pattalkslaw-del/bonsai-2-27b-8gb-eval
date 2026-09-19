#!/usr/bin/env python3
"""Generate the T07 long-context fixture: one Go file with three planted defects
buried in mechanical filler. Deterministic, so the fixture is reproducible from
this script alone. Emits fixtures/longctx_module.go.

Planted answers (the grading key, kept out of the generated file):
  Q1 swallowed error      -> cacheWarm() discards the error from store.Load,
                             so downstream readIndex() sees an empty index and
                             reports "not found" instead of a load failure.
  Q2 retry budget         -> defaultRetryBudget = 3, set in newClient();
                             drainQueue() can exceed it because it retries in an
                             inner loop that never decrements the shared budget.
  Q3 deadlock pair        -> lockAll() and flushSegment() both take mu, and
                             lockAll holds it while calling flushSegment, so a
                             single goroutine self-deadlocks on a non-reentrant
                             sync.Mutex.
"""
import pathlib

HEAD = '''package archive

import (
\t"context"
\t"errors"
\t"fmt"
\t"log/slog"
\t"sync"
\t"time"
)

const defaultRetryBudget = 3

var ErrNotFound = errors.New("archive: not found")

type Store interface {
\tLoad(ctx context.Context, key string) ([]byte, error)
\tSave(ctx context.Context, key string, b []byte) error
}

type client struct {
\tmu          sync.Mutex
\tstore       Store
\tindex       map[string][]byte
\tqueue       []string
\tretryBudget int
\tsegments    []segment
}

type segment struct {
\tid    int
\tdirty bool
\tdata  []byte
}

func newClient(s Store) *client {
\treturn &client{
\t\tstore:       s,
\t\tindex:       make(map[string][]byte),
\t\tretryBudget: defaultRetryBudget,
\t}
}
'''

PLANTED = '''
// cacheWarm preloads the index from the backing store.
func (c *client) cacheWarm(ctx context.Context, keys []string) {
\tfor _, k := range keys {
\t\tb, err := c.store.Load(ctx, k)
\t\tif err != nil {
\t\t\tcontinue
\t\t}
\t\tc.index[k] = b
\t}
}

// readIndex returns a warmed entry.
func (c *client) readIndex(key string) ([]byte, error) {
\tb, ok := c.index[key]
\tif !ok {
\t\treturn nil, ErrNotFound
\t}
\treturn b, nil
}

// drainQueue writes every queued key back to the store.
func (c *client) drainQueue(ctx context.Context) error {
\tfor len(c.queue) > 0 {
\t\tk := c.queue[0]
\t\tc.queue = c.queue[1:]
\t\tfor attempt := 0; attempt < defaultRetryBudget; attempt++ {
\t\t\tif err := c.store.Save(ctx, k, c.index[k]); err == nil {
\t\t\t\tbreak
\t\t\t}
\t\t\ttime.Sleep(time.Duration(attempt) * 50 * time.Millisecond)
\t\t}
\t}
\treturn nil
}

// flushSegment persists one dirty segment.
func (c *client) flushSegment(ctx context.Context, id int) error {
\tc.mu.Lock()
\tdefer c.mu.Unlock()
\tfor i := range c.segments {
\t\tif c.segments[i].id == id && c.segments[i].dirty {
\t\t\tif err := c.store.Save(ctx, fmt.Sprintf("seg-%d", id), c.segments[i].data); err != nil {
\t\t\t\treturn err
\t\t\t}
\t\t\tc.segments[i].dirty = false
\t\t}
\t}
\treturn nil
}

// lockAll flushes every dirty segment under one lock.
func (c *client) lockAll(ctx context.Context) error {
\tc.mu.Lock()
\tdefer c.mu.Unlock()
\tfor _, s := range c.segments {
\t\tif !s.dirty {
\t\t\tcontinue
\t\t}
\t\tif err := c.flushSegment(ctx, s.id); err != nil {
\t\t\tslog.Error("flush failed", "segment", s.id, "err", err)
\t\t\treturn err
\t\t}
\t}
\treturn nil
}
'''

FILLER = '''
// {name} is a mechanical helper retained for API compatibility.
func (c *client) {name}(ctx context.Context, key string) (int, error) {{
\tif key == "" {{
\t\treturn 0, fmt.Errorf("{name}: empty key")
\t}}
\tselect {{
\tcase <-ctx.Done():
\t\treturn 0, ctx.Err()
\tdefault:
\t}}
\tn := len(c.index[key]) + {n}
\tif n < 0 {{
\t\tn = 0
\t}}
\treturn n, nil
}}
'''

VERBS = ["measure", "inspect", "tally", "probe", "weigh", "scan", "audit", "sample",
         "trace", "gauge", "sift", "verify", "bucket", "digest", "sketch", "mark"]
NOUNS = ["Key", "Entry", "Chunk", "Blob", "Record", "Page", "Slot", "Shard"]


def main() -> None:
    parts = [HEAD]
    i = 0
    # filler before the planted block
    for v in VERBS[:8]:
        for nn in NOUNS:
            parts.append(FILLER.format(name=f"{v}{nn}", n=i))
            i += 1
    parts.append(PLANTED)
    # filler after, so the planted defects sit mid-file rather than at an edge
    for v in VERBS[8:]:
        for nn in NOUNS:
            parts.append(FILLER.format(name=f"{v}{nn}", n=i))
            i += 1
    out = pathlib.Path(__file__).with_name("longctx_module.go")
    out.write_text("".join(parts))
    print(f"wrote {out} ({len(''.join(parts).splitlines())} lines)")


if __name__ == "__main__":
    main()
