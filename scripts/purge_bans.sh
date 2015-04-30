#!/bin/sh
# Removes all bans
# Run from the meguca root directory
NODE=`which node`

tmp=/tmp/purge_bans_list
redis-cli keys "hot:timeouts" | xargs redis-cli smembers > $tmp
while read timeip; do
  redis-cli keys "ip:${timeip}" | xargs redis-cli del
done < $tmp
redis-cli keys "hot:timeouts" | xargs redis-cli del
redis-cli keys "auditLog" | xargs redis-cli del

$NODE server/kill.js
