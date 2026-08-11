#!/bin/sh
#
# backup.sh — pg_dump one shot, or on a loop as the compose `backup` service.
#
#   ./backup.sh            take one dump and exit
#   ./backup.sh --loop     take one immediately, then every BACKUP_INTERVAL_SECONDS
#
# Runs inside a postgres:16-alpine container, so this is POSIX sh, not bash, and
# the pg_* tools match the server version. Connection settings come from the
# standard libpq variables (PGHOST/PGUSER/PGPASSWORD/PGDATABASE) that the
# compose file already sets.
#
# Dumps are custom format (-Fc): compressed, and restorable selectively with
# pg_restore. Plain SQL would be larger and offer no way to restore one table.
#
# Take one of these by hand before every migration — see deployments/README.md.
set -eu

BACKUP_DIR="${BACKUP_DIR:-/backup/dumps}"
BACKUP_RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-14}"
BACKUP_INTERVAL_SECONDS="${BACKUP_INTERVAL_SECONDS:-86400}"

log() { echo "[backup] $(date -u +%Y-%m-%dT%H:%M:%SZ) $*"; }

take_dump() {
	mkdir -p "$BACKUP_DIR"

	stamp="$(date -u +%Y%m%dT%H%M%SZ)"
	target="${BACKUP_DIR}/${PGDATABASE}-${stamp}.dump"

	# Write to a .partial name first and rename on success. A dump interrupted
	# midway would otherwise sit in the directory looking exactly like a good
	# one, and be found by whoever is restoring under pressure.
	log "dumping ${PGDATABASE} from ${PGHOST}"
	if ! pg_dump --format=custom --compress=9 --file="${target}.partial"; then
		log "ERROR: pg_dump failed"
		rm -f "${target}.partial"
		return 1
	fi
	mv "${target}.partial" "$target"

	size="$(du -h "$target" | cut -f1)"
	log "wrote ${target} (${size})"

	# Retention runs only after a dump succeeds, so a broken backup job cannot
	# quietly age out the last good copies.
	deleted="$(find "$BACKUP_DIR" -name "${PGDATABASE}-*.dump" -type f \
		-mtime "+${BACKUP_RETENTION_DAYS}" -print -delete | wc -l | tr -d ' ')"
	if [ "$deleted" != "0" ]; then
		log "pruned ${deleted} dump(s) older than ${BACKUP_RETENTION_DAYS} days"
	fi
}

case "${1:-}" in
--loop)
	log "starting: every ${BACKUP_INTERVAL_SECONDS}s, keeping ${BACKUP_RETENTION_DAYS} days"
	while true; do
		# A failed dump must not kill the service, or one transient outage ends
		# all future backups until somebody notices.
		take_dump || log "continuing after failure; next attempt in ${BACKUP_INTERVAL_SECONDS}s"
		sleep "$BACKUP_INTERVAL_SECONDS"
	done
	;;
"")
	take_dump
	;;
*)
	echo "usage: $0 [--loop]" >&2
	exit 2
	;;
esac
