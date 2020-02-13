#!/usr/bin/env bash

# md5 for user `all` is enabled by default if POSTGRES_PASSWORD env var is present.
echo "host all all 0.0.0.0/0 trust" >> "${PGDATA}/pg_hba.conf"
echo "local all all trust" >> "${PGDATA}/pg_hba.conf"

echo "listen_addresses='*'" >> "${PGDATA}/postgresql.conf"

pg_ctl restart
