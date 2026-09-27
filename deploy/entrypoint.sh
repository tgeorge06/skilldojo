#!/bin/sh
# Production (Fly): restore the database from the replica if the volume is
# empty, then run the app under Litestream so every change streams to the
# bucket. Replication is mandatory unless LITESTREAM_DISABLED=1 is set
# explicitly (local Docker only); a missing secret must not silently run the
# app with no backups.
set -eu
DATABASE_PATH="${DATABASE_PATH:-/data/skilldojo.db}"
export DATABASE_PATH
mkdir -p "$(dirname "$DATABASE_PATH")"
chown -R app:app "$(dirname "$DATABASE_PATH")"

if [ "${LITESTREAM_DISABLED:-0}" = "1" ]; then
  echo "litestream disabled by LITESTREAM_DISABLED=1; running without replication" >&2
  exec su-exec app /app/skilldojo -addr 0.0.0.0:8080 -db "$DATABASE_PATH"
fi
for v in LITESTREAM_BUCKET LITESTREAM_ENDPOINT LITESTREAM_ACCESS_KEY_ID LITESTREAM_SECRET_ACCESS_KEY; do
  eval "val=\${$v:-}"
  if [ -z "$val" ]; then
    echo "refusing to start: $v is not set (set LITESTREAM_DISABLED=1 to run without backups)" >&2
    exit 1
  fi
done
su-exec app litestream restore -if-db-not-exists -if-replica-exists -config /etc/litestream.yml "$DATABASE_PATH"
exec su-exec app litestream replicate -config /etc/litestream.yml \
  -exec "/app/skilldojo -addr 0.0.0.0:8080 -db $DATABASE_PATH"
