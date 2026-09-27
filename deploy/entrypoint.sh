#!/bin/sh
# With LITESTREAM_BUCKET set, restore the database from the replica if the
# volume is empty, then run the app under Litestream so every change streams
# to the bucket. Without it, run the app directly (local Docker).
set -eu
DB="${DATABASE_PATH:-/data/skilldojo.db}"
mkdir -p "$(dirname "$DB")"
if [ -n "${LITESTREAM_BUCKET:-}" ]; then
  litestream restore -if-db-not-exists -if-replica-exists -config /etc/litestream.yml "$DB"
  exec litestream replicate -config /etc/litestream.yml -exec "/app/skilldojo -addr 0.0.0.0:8080 -db $DB"
fi
exec /app/skilldojo -addr 0.0.0.0:8080 -db "$DB"
