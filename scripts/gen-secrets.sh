#!/usr/bin/env bash
#
# gen-secrets.sh — print a deployments/.env.prod with fresh secrets.
#
#   ./scripts/gen-secrets.sh > deployments/.env.prod
#
# Takes deployments/.env.prod.example as the template and fills in the three
# empty secret values, leaving every comment and every other setting intact —
# so the generated file stays the documented one and does not drift from the
# example as settings are added.
#
# Writes to stdout rather than to the file directly: that makes it safe to run
# twice, and impossible to overwrite a live .env.prod by accident. Regenerating
# JWT_SECRET signs out every existing session; regenerating POSTGRES_PASSWORD
# against an existing database locks the stack out of its own data.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEMPLATE="$ROOT_DIR/deployments/.env.prod.example"
TARGET="$ROOT_DIR/deployments/.env.prod"

if [[ ! -f "$TEMPLATE" ]]; then
	echo "template not found: $TEMPLATE" >&2
	exit 1
fi

if [[ -t 1 ]]; then
	echo "This writes to stdout. Redirect it:" >&2
	echo "  ./scripts/gen-secrets.sh > deployments/.env.prod" >&2
	exit 2
fi

if [[ -f "$TARGET" ]]; then
	echo "warning: $TARGET already exists." >&2
	echo "         Replacing its secrets invalidates all sessions and locks the" >&2
	echo "         stack out of the existing database. Move it aside first if" >&2
	echo "         you meant to keep it." >&2
fi

# base64 of 36 random bytes: 48 characters, comfortably over the 32-character
# minimum the backend enforces, and free of the shell-hostile characters that
# would need quoting inside an env file.
secret() {
	LC_ALL=C openssl rand -base64 36 | tr -d '\n' | tr '+/' '-_'
}

JWT_SECRET="$(secret)"
POSTGRES_PASSWORD="$(secret)"
REDIS_PASSWORD="$(secret)"

# Only the three empty assignments are touched. Anchored to start-of-line so a
# mention of the name inside a comment is left alone.
sed \
	-e "s|^JWT_SECRET=$|JWT_SECRET=${JWT_SECRET}|" \
	-e "s|^POSTGRES_PASSWORD=$|POSTGRES_PASSWORD=${POSTGRES_PASSWORD}|" \
	-e "s|^REDIS_PASSWORD=$|REDIS_PASSWORD=${REDIS_PASSWORD}|" \
	"$TEMPLATE"

echo "" >&2
echo "Generated. Still to set by hand: APP_DOMAIN, ACME_EMAIL, IMAGE_TAG." >&2
echo "Then: chmod 600 deployments/.env.prod" >&2
