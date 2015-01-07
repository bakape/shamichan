#!/bin/bash
# Purges all active moderator and admin sessions,
# removing old mods and forcing everyone else to
# relogin
# Run from the meguca root directory
NODE=`which node`

for i in `redis-cli keys session:\*`; do
	redis-cli del $i
done

$NODE server/kill.js
