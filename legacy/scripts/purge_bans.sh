#!/bin/sh
# Removes all bans
# Run from the meguca root directory

redis-cli del bans
redis-cli publish cache '[3]'
