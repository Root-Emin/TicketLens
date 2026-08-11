# TicketLens — Production Deployment

Single-host deployment: Caddy terminates TLS, the Next.js frontend is the only
service the public reaches, and the Go backend, classifier, Postgres and Redis
sit on a compose network that is not routed off the host.

This is **not** the development stack. `./start.sh` and
`backend/deployments/docker-compose.yml` are unchanged and unrelated; nothing
here affects them.

```
internet ──443──> caddy ──> frontend ──> backend ──> postgres
                                    │            └──> redis
                                    │            └──> classifier
                                    └─ /api/proxy/* forwarded server-side
                                       with the JWT from an httpOnly cookie
```

The backend is deliberately not published. The browser never calls it; the
frontend proxies every API request from the server side, so there is no second
public authentication surface to defend.

---

## Requirements

- A host with Docker Engine 24+ and the compose plugin
- Ports 80 and 443 free and reachable from the internet (ACME needs both)
- A DNS **A record for `APP_DOMAIN` already pointing at the host** — Caddy
  cannot obtain a certificate before this resolves
- Read access to the GHCR packages (`docker login ghcr.io` with a PAT that has
  `read:packages`, unless the packages are public)

---

## First install

```bash
git clone <repo> ticketlens && cd ticketlens

# 1. Secrets. Writes to stdout so it can never clobber a live file.
./scripts/gen-secrets.sh > deployments/.env.prod
chmod 600 deployments/.env.prod

# 2. Set APP_DOMAIN, ACME_EMAIL and IMAGE_TAG by hand.
$EDITOR deployments/.env.prod

# 3. Bring it up. The migrate service runs to completion first; the backend
#    only starts once it has succeeded.
docker compose -f deployments/docker-compose.prod.yml \
               --env-file deployments/.env.prod up -d
```

The compose command is long enough to be worth an alias for the rest of this
document:

```bash
alias tl='docker compose -f deployments/docker-compose.prod.yml --env-file deployments/.env.prod'
```

### Verify the install

```bash
tl ps                       # every service Up; migrate shows Exited (0)
tl logs caddy | grep -i certificate
curl -sI https://$APP_DOMAIN | head -1        # 200

# The backend from inside the network — it has no public address.
tl exec backend wget -qO- http://127.0.0.1:8080/health/ready

# Which classifier is actually serving. This is the one that catches a bad
# release quietly: StubBackend means the image shipped with no checkpoint and
# every ticket is being triaged by keyword matching.
tl exec classifier python -c \
  "import urllib.request;print(urllib.request.urlopen('http://127.0.0.1:8091/healthz').read())"

# Operational endpoints must not be public.
curl -s -o /dev/null -w '%{http_code}\n' https://$APP_DOMAIN/metrics   # 404
```

---

## Upgrading

The order matters. Migrations are forward-only — there is no `down` path in the
runner — so the backup is what makes the step reversible.

```bash
# 1. Back up FIRST. Not optional on any release that changes the schema, and
#    cheap enough that it is not worth deciding whether this one does.
tl exec backup /backup/backup.sh

# 2. Point at the new release and fetch it.
$EDITOR deployments/.env.prod          # IMAGE_TAG=v0.3.0
tl pull

# 3. Apply the schema change on its own, before anything serves the new code.
tl run --rm migrate

# 4. Roll the services.
tl up -d

# 5. Confirm.
tl exec backend wget -qO- http://127.0.0.1:8080/health/ready
```

If step 3 fails, stop. The previous version is still serving correctly against
the unchanged schema; put `IMAGE_TAG` back and investigate.

### Rolling back

**No schema change in the release:** set `IMAGE_TAG` back and `tl up -d`.

**Schema changed:** the old binaries do not know the new schema, and nothing
un-applies a migration. Restore the pre-upgrade dump:

```bash
tl stop backend frontend
tl run --rm --entrypoint /bin/sh backup \
   /backup/restore.sh /backup/dumps/<dump-taken-in-step-1> --yes-i-am-sure
$EDITOR deployments/.env.prod          # IMAGE_TAG back to the previous release
tl up -d
```

Everything written between the dump and the rollback is lost. That is the cost
of a forward-only migration runner, and the reason step 1 exists.

---

## Backups

The `backup` service dumps nightly into `deployments/backup/dumps` on the host
(`BACKUP_INTERVAL_SECONDS`, `BACKUP_RETENTION_DAYS`). Dumps are `pg_dump`
custom format.

```bash
tl exec backup /backup/backup.sh      # on demand
ls -lht deployments/backup/dumps
```

Two things this setup does **not** do, and that a real deployment must:

1. **Copy the dumps off the host.** They currently sit on the same disk as the
   database they protect. Sync `deployments/backup/dumps` to object storage.
2. **Prove a dump restores.** A backup nobody has restored is a hypothesis. Run
   the drill in *Verification* below on a scratch host before relying on it.

`caddy_data` deserves a copy too — it holds the ACME account key and issued
certificates, and Let's Encrypt rate-limits re-issuance.

---

## Configuration

All settings live in `deployments/.env.prod`; see
[.env.prod.example](.env.prod.example) for what each one does. Notes on the
ones that are easy to get wrong:

| Variable | Note |
|---|---|
| `IMAGE_TAG` | Applies to all three images at once. Never mix tags across services. |
| `JWT_SECRET` | Changing it signs every user out. Minimum 32 chars, enforced at boot. |
| `DB_SSLMODE` | `disable` is fine while Postgres is in this stack. Set `require` the moment `DB_HOST` leaves it — the backend warns about this at every startup. |
| `POSTGRES_PASSWORD` | Changing it after first install does **not** change the password inside the existing volume; the stack will simply fail to authenticate. |

### Moving to a managed database

Point `DB_HOST`/`DB_PORT` at the instance, set `DB_SSLMODE=require`, and drop
the `postgres` service from the compose file. Nothing in the application code
changes — the connection is built from environment variables
(`internal/shared/config`). Migrate the data with the same `pg_dump`/`pg_restore`
pair the backup scripts use.

---

## Troubleshooting

**Backend keeps restarting, logs `pending migration(s)`** — the migrate step was
skipped. Run `tl run --rm migrate`. The server refuses to serve against a schema
older than the binary rather than fail one query at a time under live traffic.

**Backend exits with `connect to postgres (required for APP_ENV=production)`** —
working as intended. Outside development a database-less server would answer
every request with a 500 while its readiness probe had no way to say so. Check
`tl logs postgres`.

**Caddy cannot get a certificate** — almost always DNS or a blocked port 80.
`dig +short $APP_DOMAIN` must return this host, and the ACME HTTP challenge
needs 80 open even though the site itself serves on 443.

**`/healthz` reports `StubBackend`** — the classifier image was built without a
checkpoint in `backend/ml/models`. Triage still works, by keyword matching. Fix
the release rather than the running container: the model belongs to the image
tag.

---

## Verification drills

Worth running once on a scratch host before the first customer, and after any
change to this directory.

**Readiness tells the truth**

```bash
tl stop postgres
tl exec backend wget -qO- http://127.0.0.1:8080/health/ready   # 503, postgres: unhealthy
tl start postgres
```

**Migration is idempotent**

```bash
tl run --rm migrate     # applies
tl run --rm migrate     # "schema is up to date, nothing to apply"
tl run --rm --entrypoint /app/migrate backend -status    # lists nothing
```

**A restore actually restores** — the only test that matters for a backup:

```bash
tl exec backup /backup/backup.sh          # dump
# create a ticket through the UI
tl stop backend frontend
tl run --rm --entrypoint /bin/sh backup \
   /backup/restore.sh /backup/dumps/<that-dump> --yes-i-am-sure
tl up -d
# the ticket is gone; everything older is intact
```
