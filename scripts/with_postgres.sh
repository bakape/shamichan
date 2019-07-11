#!/usr/bin/env bash
# Wait until PostgreSQL is started and then execute the arguments

echo "Waiting for PostgreSQL server..."
until nc -z postgres 5432
do
    sleep 1
done

eval $@
