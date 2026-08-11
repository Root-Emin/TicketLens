#!/bin/sh
#
# restore.sh — replace the current database contents with a dump.
#
#   ./restore.sh /backup/dumps/ticketlens-20260808T030000Z.dump --yes-i-am-sure
#
# THIS IS DESTRUCTIVE. Everything written since that dump is gone. It is the
# only way back from a bad migration (the migration runner is forward-only), so
# it exists — but it will not run without the confirmation flag.
#
# Intended to be run inside the stack, with the application stopped:
#
#   docker compose -f deployments/docker-compose.prod.yml stop backend frontend
#   docker compose -f deployments/docker-compose.prod.yml run --rm \
#       --entrypoint /bin/sh backup /backup/restore.sh <dump> --yes-i-am-sure
#   docker compose -f deployments/docker-compose.prod.yml up -d
#
# Stopping the backend first is not optional: restoring underneath a running
# server means live requests read a schema that is being dropped and recreated.
set -eu

dump="${1:-}"
confirm="${2:-}"

log() { echo "[restore] $(date -u +%Y-%m-%dT%H:%M:%SZ) $*"; }

if [ -z "$dump" ]; then
	echo "usage: $0 <dump-file> --yes-i-am-sure" >&2
	echo "" >&2
	echo "available dumps:" >&2
	ls -1t "${BACKUP_DIR:-/backup/dumps}" 2>/dev/null >&2 || echo "  (none)" >&2
	exit 2
fi

if [ ! -f "$dump" ]; then
	echo "no such dump: $dump" >&2
	exit 1
fi

if [ "$confirm" != "--yes-i-am-sure" ]; then
	echo "refusing to restore $dump without --yes-i-am-sure" >&2
	echo "this drops the current contents of ${PGDATABASE} on ${PGHOST}" >&2
	exit 1
fi

log "restoring ${PGDATABASE} on ${PGHOST} from ${dump}"

# --clean --if-exists drops each object before recreating it, so the restore
# does not require an empty database. Applied inside a single transaction so a
# failure partway leaves the database as it was rather than half-replaced.
pg_restore \
	--dbname="$PGDATABASE" \
	--clean \
	--if-exists \
	--single-transaction \
	--no-owner \
	--no-privileges \
	"$dump"

log "restore complete"
log "the schema is now whatever that dump contained — if it predates the"
log "running release, start the migrate service before the backend."
