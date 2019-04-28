#!/usr/bin/env bash
# Start Postgres inside Docker and execute the arguments

service postgresql start
echo "Waiting on PostgreSQL server..."
until pg_isready > /dev/null
do
    sleep 1
done

eval $@
