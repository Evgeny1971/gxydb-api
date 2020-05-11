#!/usr/bin/env bash
# run misc/build.sh from project root
set -e
set -x

docker image build -t gxydb-api:latest .
docker create --name dummy gxydb-api:latest
docker cp dummy:/app/gxydb-api ./gxydb-api-linux
docker rm -f dummy
