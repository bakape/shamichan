#!/bin/sh
# Removes all bans
# Make sure to do a server/kill.js after running this
# So that the server can reset from the changes

tmp=/tmp/purge_bans_list
redis-cli keys "hot:timeouts" | xargs redis-cli smembers > $tmp
while read timeip; do
  redis-cli keys "ip:${timeip}" | xargs redis-cli del
done < $tmp
redis-cli keys "hot:timeouts" | xargs redis-cli del
redis-cli keys "auditLog" | xargs redis-cli del
