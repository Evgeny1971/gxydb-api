#!/usr/bin/env bash
set -e

WORKDIR=/opt/gxydb

export $(cat "$WORKDIR/.env" | xargs)

echo "sessions count"
psql $DB_URL -c 'select count(*) from sessions;'

echo "truncate tables"
psql $DB_URL -c 'truncate sessions RESTART IDENTITY;'

echo "restart process"
systemctl restart gxydb
