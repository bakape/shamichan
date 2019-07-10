#!/usr/bin/env bash
# Start Postgres inside Docker and execute the arguments

echo "Waiting on PostgreSQL server..."
until nc -z postgres 5432
do
    sleep 1
done

eval $@
