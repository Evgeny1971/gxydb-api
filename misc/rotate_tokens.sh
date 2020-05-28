#!/usr/bin/env bash
set -e

BASE_DIR="/opt/gxydb"
TIMESTAMP="$(date '+%Y%m%d%H%M%S')"
LOG_FILE="$BASE_DIR/logs/rotate_tokens_$TIMESTAMP.log"

cd ${BASE_DIR}
./gxydb-api-linux rotate_tokens --max-age=7 >>${LOG_FILE} 2>&1

HAS_ERRORS="$(jq -r -c 'select(.level == "error" or .level == "warning" or .level == "fatal")' ${LOG_FILE} | wc -l)"

find ${BASE_DIR}/logs -name "rotate_tokens_*" -type f -mtime +7 -exec rm -f {} \;

if [ "${HAS_ERRORS}" = "0" ]; then
  echo "Sanity OK."
  exit 0
else
  echo "gxydb rotate_tokens error" | mail -s "ERROR: gxydb rotate_tokens." -r "gxydb@galaxy.kli.one" -a ${LOG_FILE} edoshor@gmail.com
  exit 1
fi
