#!/usr/bin/env bash
# Docker Travis testing script

service postgresql start
echo "Waiting on PostgreSQL server..."
until pg_isready > /dev/null
do
    sleep 1
done

. /emsdk/emsdk_env.sh
make test # TODO: Reenable for wasm library: wasm
