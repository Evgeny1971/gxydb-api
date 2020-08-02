#!/usr/bin/env bash
# run misc/build.sh from project root
set -e
set -x

docker image build -t gxydb-api:latest .
version="$(docker run --rm gxydb-api:latest /app/gxydb-api version | awk '{print $NF}')"
docker create --name dummy gxydb-api:latest
docker cp dummy:/app/gxydb-api ./gxydb-api-linux-"${version}"
docker rm -f dummy
