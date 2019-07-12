#!/bin/bash

# Increase PostgreSQL connection limit by changing `max_connections` to 1024
sed -i "/max_connections =/d" /var/lib/postgresql/data/postgresql.conf
echo max_connections = 1024 >> /var/lib/postgresql/data/postgresql.conf
