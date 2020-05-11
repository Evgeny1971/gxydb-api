#!/usr/bin/env bash
# run misc/deploy.sh from project root
set -e
set -x

scp gxydb-api-linux root@gxydb.kli.one:/opt/gxydb/gxydb-api-linux.new
scp -r migrations root@gxydb.kli.one:/opt/gxydb

ssh root@gxydb.kli.one "cd /opt/gxydb && export \$(cat .env | xargs) && ./migrate -database \$DB_URL -path migrations up"
ssh root@gxydb.kli.one "/bin/cp -f /opt/gxydb/gxydb-api-linux /opt/gxydb/gxydb-api-linux.old"
ssh root@gxydb.kli.one "systemctl stop gxydb"
ssh root@gxydb.kli.one "mv /opt/gxydb/gxydb-api-linux.new /opt/gxydb/gxydb-api-linux"
ssh root@gxydb.kli.one "systemctl start gxydb"
