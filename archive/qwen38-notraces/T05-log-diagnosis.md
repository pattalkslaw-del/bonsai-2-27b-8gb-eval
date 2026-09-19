# T05 log-diagnosis (qwen38)

finish_reason: stop  elapsed: 500.31s  usage: {"prompt_tokens": 397, "total_tokens": 34559, "completion_tokens": 34162, "prompt_tokens_details": null}

## Answer



## Root cause

On first boot the official PostgreSQL Docker image does not just start the final PostgreSQL server. It:

1. initializes the data directory,
2. starts a **temporary** PostgreSQL server to run `/docker-entrypoint-initdb.d` scripts,
3. stops that temporary server,
4. then starts the real/final PostgreSQL server.

Your log is showing that temporary server being stopped:

```text
02:14:07.402 db   the database system is shutting down
```

Then the entrypoint finishes initialization and starts the real server:

```text
02:14:08.990 db   PostgreSQL init process complete; ready for start up
02:14:09.140 db   database system is ready to accept connections
```

The app started at:

```text
02:14:07.118 app  starting migrations
```

so it connected to the **temporary** PostgreSQL instance. That instance was then stopped, so `pg` reported:

```text
Error: Connection terminated unexpectedly
```

The retry works because on the second start the data directory is already initialized, so the image does not start a temporary server. The first server started is the final one, so the healthcheck is then accurate.

## Why the healthcheck passes anyway

This healthcheck:

```yaml
test: ["CMD-SHELL", "pg_isready -U app"]
```

only checks that *some* PostgreSQL server is accepting connections. It does not know whether that server is the temporary initialization server or the final server.

During first boot, the temporary server is a real PostgreSQL server and is accepting connections, so `pg_isready` exits `0`. Docker Compose therefore marks `db` healthy and starts `app` too early.

Also, `pg_isready -U app` without `-d` uses the database named `app` by default. In the standard PostgreSQL image, if `POSTGRES_USER=app`, the `app` database is created during initialization, so the temporary server already satisfies the healthcheck.

## Minimal corrected Compose fragment

For the standard `postgres:16` image, the temporary server is started with a `config_file=` argument, while the final server is started as plain `postgres`. The running postmaster records its command line in `$PGDATA/postmaster.opts`, so you can use that to distinguish the temporary server from the final server.

```yaml
services:
  db:
    image: postgres:16
    # keep your existing environment/volumes
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U app -d app && test -f \"$PGDATA/postmaster.opts\" && ! grep -q -- '-c config_file=' \"$PGDATA/postmaster.opts\""]
      interval: 5s
      timeout: 5s
      retries: 3
      start_period: 30s

  app:
    # existing app service
    depends_on:
      db:
        condition: service_healthy
```

Use the same user/database in `pg_isready` that your app actually connects to. If your app uses a different database, replace `app` with that database name.

The important parts:

- `pg_isready -U app -d app` checks that a server is accepting connections.
- `! grep -q -- '-c config_file=' "$PGDATA/postmaster.opts"` rejects the temporary initialization server.
- `start_period: 30s` prevents the expected failures during first initialization from counting against `retries`. Increase it if your init scripts are slow.

This assumes you are using the standard image’s default `postgres` command. If you override the PostgreSQL command in a way that also passes `config_file=` to the final server, adjust the marker accordingly.

## Application-side change

Strictly, no application change is required once the healthcheck is fixed.

However, a small bounded retry is still a good safety net, especially for later database restarts or network blips:

```js
const { Pool } = require('pg');

const pool = new Pool({
  // your existing pg config
});

const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

async function waitForDatabase(pool, attempts = 10) {
  for (let i = 1; i <= attempts; i++) {
    try {
      await pool.query('SELECT 1');
      return;
    } catch (err) {
      if (i === attempts) {
        throw err;
      }
      await sleep(250 * i);
    }
  }
}

await waitForDatabase(pool);
await runMigrations(pool);
```

If you use a single `Client` instead of a `Pool`, retry `client.connect()` in the same way.

