#!/usr/bin/env bash
# Docker entry point script

service postgresql start
echo "Waiting on PostgreSQL server..."
until pg_isready > /dev/null
do
    sleep 1
done

make test
